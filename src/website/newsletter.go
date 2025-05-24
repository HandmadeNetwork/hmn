package website

import (
	"html/template"

	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

func NewsletterSignup(c *RequestContext) ResponseData {
	var res ResponseData

	subforumTree := models.GetFullSubforumTree(c, c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)

	type TemplateData struct {
		templates.BaseData
		LatestNewsPost      *templates.TimelineItem
		NewsletterSignupUrl string
	}

	tmpl := TemplateData{
		BaseData:            getBaseData(c, "Newsletter Signup", nil),
		NewsletterSignupUrl: hmnurl.BuildAPINewsletterSignup(),
	}
	tmpl.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Newsletter Signup"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:description", Value: "Sign up for the Handmade Network email newsletter to keep up with the Handmade community."},
		{Property: "og:url", Value: hmnurl.BuildNewsletterSignup()},
	}

	newsThreads, err := hmndata.FetchThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
		ProjectIDs:     []int{models.HMNProjectID},
		ThreadTypes:    []models.ThreadType{models.ThreadTypeProjectBlogPost},
		Limit:          1,
		OrderByCreated: true,
	})
	if err == nil {
		if len(newsThreads) > 0 {
			t := newsThreads[0]
			item := PostToTimelineItem(c.UrlContext, lineageBuilder, &t.FirstPost, &t.Thread, t.FirstPostAuthor)
			item.Breadcrumbs = nil
			item.TypeTitle = ""
			item.Description = template.HTML(t.FirstPostCurrentVersion.TextParsed)
			item.AllowTitleWrap = true
			item.TruncateDescription = true
			item.Unread = t.Unread
			tmpl.LatestNewsPost = &item
		}
	} else {
		c.Logger.Error().Err(err).Msg("failed to fetch latest news post for newsletter page")
	}

	res.MustWriteTemplate("newsletter.html", tmpl, c.Perf)
	return res
}
