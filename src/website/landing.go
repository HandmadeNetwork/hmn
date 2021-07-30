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
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get projects for home page"))
	}
	defer iterProjects.Close()

	var pageProjects []LandingPageProject

	allProjects := iterProjects.ToSlice()
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	var currentUserId *int
	if c.CurrentUser != nil {
		currentUserId = &c.CurrentUser.ID
	}

	c.Perf.StartBlock("LANDING", "Process projects")
	for _, projRow := range allProjects {
		proj := projRow.(*models.Project)

		c.Perf.StartBlock("SQL", fmt.Sprintf("Fetch posts for %s", proj.Name))
		type projectPostQuery struct {
			Post                 models.Post   `db:"post"`
			Thread               models.Thread `db:"thread"`
			User                 models.User   `db:"auth_user"`
			ThreadLastReadTime   *time.Time    `db:"tlri.lastread"`
			SubforumLastReadTime *time.Time    `db:"slri.lastread"`
		}
		projectPostIter, err := db.Query(c.Context(), c.Conn, projectPostQuery{},
			`
			SELECT $columns
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
				LEFT JOIN handmade_threadlastreadinfo AS tlri ON (
					tlri.thread_id = post.thread_id
					AND tlri.user_id = $1
				)
				LEFT JOIN handmade_subforumlastreadinfo AS slri ON (
					slri.subforum_id = thread.subforum_id
					AND slri.user_id = $1
				)
				LEFT JOIN auth_user ON post.author_id = auth_user.id
			WHERE
				post.project_id = $2
				AND post.thread_type = ANY ($3)
				AND post.deleted = FALSE
			ORDER BY postdate DESC
			LIMIT $4
			`,
			currentUserId,
			proj.ID,
			[]models.ThreadType{models.ThreadTypeProjectArticle, models.ThreadTypeForumPost},
			maxPosts,
		)
		c.Perf.EndBlock()
		if err != nil {
			c.Logger.Error().Err(err).Msg("failed to fetch project posts")
			continue
		}
		projectPosts := projectPostIter.ToSlice()

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

		for _, projectPostRow := range projectPosts {
			projectPost := projectPostRow.(*projectPostQuery)

			hasRead := false
			if projectPost.ThreadLastReadTime != nil && projectPost.ThreadLastReadTime.After(projectPost.Post.PostDate) {
				hasRead = true
			} else if projectPost.SubforumLastReadTime != nil && projectPost.SubforumLastReadTime.After(projectPost.Post.PostDate) {
				hasRead = true
			}

			featurable := (!proj.IsHMN() &&
				projectPost.Post.ThreadType == models.ThreadTypeProjectArticle &&
				projectPost.Post.ParentID == nil &&
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
					Url:     hmnurl.BuildBlogPost(proj.Slug, projectPost.Thread.ID, projectPost.Post.ID),
					User:    templates.UserToTemplate(&projectPost.User, c.Theme),
					Date:    projectPost.Post.PostDate,
					Unread:  !hasRead,
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
						&projectPost.User,
						!hasRead,
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
	type newsPostQuery struct {
		Post        models.Post        `db:"post"`
		PostVersion models.PostVersion `db:"ver"`
		Thread      models.Thread      `db:"thread"`
		User        models.User        `db:"auth_user"`
	}
	newsPostRow, err := db.QueryOne(c.Context(), c.Conn, newsPostQuery{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			JOIN handmade_thread AS thread ON post.thread_id = thread.id
			JOIN auth_user ON post.author_id = auth_user.id
			JOIN handmade_postversion AS ver ON post.current_id = ver.id
		WHERE
			post.project_id = $1
			AND thread.type = $2
			AND post.id = thread.first_id
			AND NOT thread.deleted
		ORDER BY post.postdate DESC
		LIMIT 1
		`,
		models.HMNProjectID,
		models.ThreadTypeProjectArticle,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch news post"))
	}
	newsPostResult := newsPostRow.(*newsPostQuery)
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
		ORDER BY snippet.when DESC
		LIMIT 20
		`,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets"))
	}
	snippetQuerySlice := snippetQueryResult.ToSlice()
	showcaseItems := make([]templates.TimelineItem, 0, len(snippetQuerySlice))
	for _, s := range snippetQuerySlice {
		row := s.(*snippetQuery)
		timelineItem := SnippetToTimelineItem(&row.Snippet, row.Asset, row.DiscordMessage, &row.Owner, c.Theme)
		if timelineItem.Type != templates.TimelineTypeSnippetYoutube {
			showcaseItems = append(showcaseItems, timelineItem)
		}
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SHOWCASE", "Convert to json")
	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)
	c.Perf.EndBlock()

	baseData := getBaseData(c)
	baseData.BodyClasses = append(baseData.BodyClasses, "hmdev", "landing") // TODO: Is "hmdev" necessary any more?

	var res ResponseData
	err = res.WriteTemplate("landing.html", LandingTemplateData{
		BaseData:    baseData,
		FeedUrl:     hmnurl.BuildFeed(),
		PodcastUrl:  hmnurl.BuildPodcast(models.HMNProjectSlug),
		StreamsUrl:  hmnurl.BuildStreams(),
		IRCUrl:      hmnurl.BuildBlogThread(models.HMNProjectSlug, 1138, "[Tutorial] Handmade Network IRC", 1),
		DiscordUrl:  "https://discord.gg/hxWxDee",
		ShowUrl:     "https://handmadedev.show/",
		ShowcaseUrl: hmnurl.BuildShowcase(),
		NewsPost: LandingPageFeaturedPost{
			Title:   newsPostResult.Thread.Title,
			Url:     hmnurl.BuildBlogPost(models.HMNProjectSlug, newsPostResult.Thread.ID, newsPostResult.Post.ID),
			User:    templates.UserToTemplate(&newsPostResult.User, c.Theme),
			Date:    newsPostResult.Post.PostDate,
			Unread:  true, // TODO
			Content: template.HTML(newsPostResult.PostVersion.TextParsed),
		},
		PostColumns:          cols,
		ShowcaseTimelineJson: showcaseJson,
	}, c.Perf)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render landing page template"))
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
