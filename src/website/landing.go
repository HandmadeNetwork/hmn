package website

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

type LandingTemplateData struct {
	templates.BaseData

	NewsPost             LandingPageFeaturedPost
	PostColumns          [][]LandingPageProject
	ShowcaseTimelineJson string

	FeedUrl     string
	PodcastUrl  string
	StreamsUrl  string
	IRCUrl      string
	DiscordUrl  string
	ShowUrl     string
	ShowcaseUrl string

	WheelJamUrl string
}

type LandingPageProject struct {
	Project      templates.Project
	FeaturedPost *LandingPageFeaturedPost
	Posts        []templates.PostListItem
	ForumsUrl    string
}

type LandingPageFeaturedPost struct {
	Title   string
	Url     string
	User    templates.User
	Date    time.Time
	Unread  bool
	Content template.HTML
}

func Index(c *RequestContext) ResponseData {
	const maxPosts = 5
	const numProjectsToGet = 7

	c.Perf.StartBlock("SQL", "Fetch projects")
	iterProjects, err := db.Query(c.Context(), c.Conn, models.Project{},
		`
		SELECT $columns
		FROM handmade_project
		WHERE
			(flags = 0 AND lifecycle = ANY($1))
			OR id = $2
		ORDER BY all_last_updated DESC
		LIMIT $3
		`,
		models.VisibleProjectLifecycles,
		models.HMNProjectID,
		numProjectsToGet*2, // hedge your bets against projects that don't have any content
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get projects for home page"))
	}
	defer iterProjects.Close()

	var pageProjects []LandingPageProject

	allProjects := iterProjects.ToSlice()
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	c.Perf.StartBlock("LANDING", "Process projects")
	for _, projRow := range allProjects {
		proj := projRow.(*models.Project)

		c.Perf.StartBlock("SQL", fmt.Sprintf("Fetch posts for %s", proj.Name))
		projectPosts, err := FetchPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
			ProjectIDs:     []int{proj.ID},
			ThreadTypes:    []models.ThreadType{models.ThreadTypeProjectBlogPost, models.ThreadTypeForumPost},
			Limit:          maxPosts,
			SortDescending: true,
		})
		c.Perf.EndBlock()
		if err != nil {
			c.Logger.Error().Err(err).Msg("failed to fetch project posts")
			continue
		}

		forumsUrl := ""
		if proj.ForumID != nil {
			forumsUrl = hmnurl.BuildForum(proj.Slug, lineageBuilder.GetSubforumLineageSlugs(*proj.ForumID), 1)
		} else {
			c.Logger.Error().Int("ProjectID", proj.ID).Str("ProjectName", proj.Name).Msg("Project fetched by landing page but it doesn't have forums")
		}

		landingPageProject := LandingPageProject{
			Project:   templates.ProjectToTemplate(proj, c.Theme),
			ForumsUrl: forumsUrl,
		}

		for _, projectPost := range projectPosts {
			featurable := (!proj.IsHMN() &&
				projectPost.Post.ThreadType == models.ThreadTypeProjectBlogPost &&
				projectPost.Thread.FirstID == projectPost.Post.ID &&
				landingPageProject.FeaturedPost == nil)

			if featurable {
				type featuredContentResult struct {
					Content string `db:"ver.text_parsed"`
				}

				c.Perf.StartBlock("SQL", "Fetch featured post content")
				contentResult, err := db.QueryOne(c.Context(), c.Conn, featuredContentResult{}, `
					SELECT $columns
					FROM
						handmade_post AS post
						JOIN handmade_postversion AS ver ON post.current_id = ver.id
					WHERE
						post.id = $1
				`, projectPost.Post.ID)
				c.Perf.EndBlock()
				if err != nil {
					c.Logger.Error().Err(err).Msg("failed to fetch featured post content")
					continue
				}
				content := contentResult.(*featuredContentResult).Content

				landingPageProject.FeaturedPost = &LandingPageFeaturedPost{
					Title:   projectPost.Thread.Title,
					Url:     hmnurl.BuildBlogThread(proj.Slug, projectPost.Thread.ID, projectPost.Thread.Title),
					User:    templates.UserToTemplate(projectPost.Author, c.Theme),
					Date:    projectPost.Post.PostDate,
					Unread:  projectPost.Unread,
					Content: template.HTML(content),
				}
			} else {
				landingPageProject.Posts = append(
					landingPageProject.Posts,
					MakePostListItem(
						lineageBuilder,
						proj,
						&projectPost.Thread,
						&projectPost.Post,
						projectPost.Author,
						projectPost.Unread,
						false,
						c.Theme,
					),
				)
			}
		}

		if len(projectPosts) > 0 {
			pageProjects = append(pageProjects, landingPageProject)
		}

		if len(pageProjects) >= numProjectsToGet {
			break
		}
	}
	c.Perf.EndBlock()

	/*
		Columns are filled by placing projects into the least full column.
		The fill array tracks the estimated sizes.

		This is all hardcoded for two columns; deal with it.
	*/
	cols := [][]LandingPageProject{nil, nil}
	fill := []int{4, 0}
	featuredIndex := []int{0, 0}
	for _, pageProject := range pageProjects {
		leastFullColumnIndex := indexOfSmallestInt(fill)

		numNewPosts := len(pageProject.Posts)
		if numNewPosts > maxPosts {
			numNewPosts = maxPosts
		}

		fill[leastFullColumnIndex] += numNewPosts

		if pageProject.FeaturedPost != nil {
			fill[leastFullColumnIndex] += 2 // featured posts add more to height

			// projects with featured posts go at the top of the column
			cols[leastFullColumnIndex] = append(cols[leastFullColumnIndex], pageProject)
			featuredIndex[leastFullColumnIndex] += 1
		} else {
			cols[leastFullColumnIndex] = append(cols[leastFullColumnIndex], pageProject)
		}
	}

	c.Perf.StartBlock("SQL", "Get news")
	newsThreads, err := FetchThreads(c.Context(), c.Conn, c.CurrentUser, ThreadsQuery{
		ProjectIDs:  []int{models.HMNProjectID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
		Limit:       1,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch news post"))
	}
	var newsThread ThreadAndStuff
	if len(newsThreads) > 0 {
		newsThread = newsThreads[0]
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetch showcase snippets")
	type snippetQuery struct {
		Owner          models.User            `db:"owner"`
		Snippet        models.Snippet         `db:"snippet"`
		Asset          *models.Asset          `db:"asset"`
		DiscordMessage *models.DiscordMessage `db:"discord_message"`
	}
	snippetQueryResult, err := db.Query(c.Context(), c.Conn, snippetQuery{},
		`
		SELECT $columns
		FROM
			handmade_snippet AS snippet
			INNER JOIN auth_user AS owner ON owner.id = snippet.owner_id
			LEFT JOIN handmade_asset AS asset ON asset.id = snippet.asset_id
			LEFT JOIN handmade_discordmessage AS discord_message ON discord_message.id = snippet.discord_message_id
		WHERE
			NOT snippet.is_jam
		ORDER BY snippet.when DESC
		LIMIT 20
		`,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets"))
	}
	snippetQuerySlice := snippetQueryResult.ToSlice()
	showcaseItems := make([]templates.TimelineItem, 0, len(snippetQuerySlice))
	for _, s := range snippetQuerySlice {
		row := s.(*snippetQuery)
		timelineItem := SnippetToTimelineItem(&row.Snippet, row.Asset, row.DiscordMessage, &row.Owner, c.Theme)
		if timelineItem.CanShowcase {
			showcaseItems = append(showcaseItems, timelineItem)
		}
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SHOWCASE", "Convert to json")
	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)
	c.Perf.EndBlock()

	baseData := getBaseData(c, "", nil)
	baseData.BodyClasses = append(baseData.BodyClasses, "hmdev", "landing") // TODO: Is "hmdev" necessary any more?
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    "A community of programmers committed to producing quality software through deeper understanding.",
	})

	var res ResponseData
	err = res.WriteTemplate("landing.html", LandingTemplateData{
		BaseData:    baseData,
		FeedUrl:     hmnurl.BuildFeed(),
		PodcastUrl:  hmnurl.BuildPodcast(models.HMNProjectSlug),
		StreamsUrl:  hmnurl.BuildStreams(),
		IRCUrl:      hmnurl.BuildBlogThread(models.HMNProjectSlug, 1138, "[Tutorial] Handmade Network IRC"),
		DiscordUrl:  "https://discord.gg/hxWxDee",
		ShowUrl:     "https://handmadedev.show/",
		ShowcaseUrl: hmnurl.BuildShowcase(),
		NewsPost: LandingPageFeaturedPost{
			Title:   newsThread.Thread.Title,
			Url:     hmnurl.BuildBlogThread(models.HMNProjectSlug, newsThread.Thread.ID, newsThread.Thread.Title),
			User:    templates.UserToTemplate(newsThread.FirstPostAuthor, c.Theme),
			Date:    newsThread.FirstPost.PostDate,
			Unread:  true,
			Content: template.HTML(newsThread.FirstPostCurrentVersion.TextParsed),
		},
		PostColumns:          cols,
		ShowcaseTimelineJson: showcaseJson,

		WheelJamUrl: hmnurl.BuildJamIndex(),
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render landing page template"))
	}

	return res
}

func indexOfSmallestInt(s []int) int {
	result := 0
	min := s[result]

	for i, val := range s {
		if val < min {
			result = i
			min = val
		}
	}

	return result
}
