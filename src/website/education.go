package website

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/parsing"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

func EducationIndex(c *RequestContext) ResponseData {
	type indexData struct {
		templates.BaseData
		Articles      []templates.EduArticle
		NewArticleUrl string
	}

	articles, err := fetchEduArticles(c, c.Conn, models.EduArticleTypeArticle, c.CurrentUser)
	if err != nil {
		panic(err)
	}

	var tmplArticles []templates.EduArticle
	for _, article := range articles {
		tmplArticles = append(tmplArticles, templates.EducationArticleToTemplate(&article))
	}

	tmpl := indexData{
		BaseData:      getBaseData(c, "Handmade Education", nil),
		Articles:      tmplArticles,
		NewArticleUrl: hmnurl.BuildEducationArticleNew(),
	}

	var res ResponseData
	res.MustWriteTemplate("education_index.html", tmpl, c.Perf)
	return res
}

func EducationGlossary(c *RequestContext) ResponseData {
	type glossaryData struct {
		templates.BaseData
	}

	tmpl := glossaryData{
		BaseData: getBaseData(c, "Handmade Education", nil),
	}

	var res ResponseData
	res.MustWriteTemplate("education_glossary.html", tmpl, c.Perf)
	return res
}

var reEduEditorsNote = regexp.MustCompile(`<span\s*class="note".*?>.*?</span>`)

func EducationArticle(c *RequestContext) ResponseData {
	type articleData struct {
		templates.BaseData
		Article   templates.EduArticle
		EditUrl   string
		DeleteUrl string
	}

	article, err := fetchEduArticle(c, c.Conn, c.PathParams["slug"], models.EduArticleTypeArticle, c.CurrentUser)
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	tmpl := articleData{
		BaseData:  getBaseData(c, article.Title, nil),
		Article:   templates.EducationArticleToTemplate(article),
		EditUrl:   hmnurl.BuildEducationArticleEdit(article.Slug),
		DeleteUrl: hmnurl.BuildEducationArticleDelete(article.Slug),
	}
	tmpl.OpenGraphItems = append(tmpl.OpenGraphItems,
		templates.OpenGraphItem{Property: "og:description", Value: string(article.Description)},
	)
	tmpl.Breadcrumbs = []templates.Breadcrumb{
		{Name: "Education", Url: hmnurl.BuildEducationIndex()},
		{Name: article.Title, Url: hmnurl.BuildEducationArticle(article.Slug)},
	}

	// Remove editor's notes
	if c.CurrentUser == nil || !c.CurrentUser.CanAuthorEducation() {
		tmpl.Article.Content = template.HTML(reEduEditorsNote.ReplaceAllLiteralString(string(tmpl.Article.Content), ""))
	}

	var res ResponseData
	res.MustWriteTemplate("education_article.html", tmpl, c.Perf)
	return res
}

func EducationArticleNew(c *RequestContext) ResponseData {
	type adminData struct {
		editorData
		Article map[string]interface{}
	}

	tmpl := adminData{
		editorData: getEditorDataForEduArticle(c.UrlContext, c.CurrentUser, getBaseData(c, "New Education Article", nil), nil),
	}
	tmpl.editorData.SubmitUrl = hmnurl.BuildEducationArticleNew()

	var res ResponseData
	res.MustWriteTemplate("editor.html", tmpl, c.Perf)
	return res
}

func EducationArticleNewSubmit(c *RequestContext) ResponseData {
	form, err := c.GetFormValues()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, err)
	}

	art, ver := getEduArticleFromForm(form)

	dupe := 0 < db.MustQueryOneScalar[int](c, c.Conn,
		`
		SELECT COUNT(*) FROM education_article
		WHERE slug = $1
		`,
		art.Slug,
	)
	if dupe {
		return c.RejectRequest("A resource already exists with that slug.")
	}

	createEduArticle(c, art, ver)

	res := c.Redirect(eduArticleURL(&art), http.StatusSeeOther)
	res.AddFutureNotice("success", "Created new education article.")
	return res
}

func EducationArticleEdit(c *RequestContext) ResponseData {
	type adminData struct {
		editorData
		Article templates.EduArticle
	}

	article, err := fetchEduArticle(c, c.Conn, c.PathParams["slug"], 0, c.CurrentUser)
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		panic(err)
	}

	tmpl := adminData{
		editorData: getEditorDataForEduArticle(c.UrlContext, c.CurrentUser, getBaseData(c, "Edit Education Article", nil), article),
		Article:    templates.EducationArticleToTemplate(article),
	}
	tmpl.editorData.SubmitUrl = hmnurl.BuildEducationArticleEdit(c.PathParams["slug"])

	var res ResponseData
	res.MustWriteTemplate("editor.html", tmpl, c.Perf)
	return res
}

func EducationArticleEditSubmit(c *RequestContext) ResponseData {
	form, err := c.GetFormValues()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, err)
	}

	_, err = fetchEduArticle(c, c.Conn, c.PathParams["slug"], 0, c.CurrentUser)
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		panic(err)
	}

	art, ver := getEduArticleFromForm(form)
	updateEduArticle(c, c.PathParams["slug"], art, ver)

	res := c.Redirect(eduArticleURL(&art), http.StatusSeeOther)
	res.AddFutureNotice("success", "Edited education article.")
	return res
}

func EducationArticleDelete(c *RequestContext) ResponseData {
	article, err := fetchEduArticle(c, c.Conn, c.PathParams["slug"], 0, c.CurrentUser)
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		panic(err)
	}

	type deleteData struct {
		templates.BaseData
		Article   templates.EduArticle
		SubmitUrl string
	}

	baseData := getBaseData(c, fmt.Sprintf("Deleting \"%s\"", article.Title), nil)

	var res ResponseData
	res.MustWriteTemplate("education_article_delete.html", deleteData{
		BaseData:  baseData,
		Article:   templates.EducationArticleToTemplate(article),
		SubmitUrl: hmnurl.BuildEducationArticleDelete(article.Slug),
	}, c.Perf)
	return res
}

func EducationArticleDeleteSubmit(c *RequestContext) ResponseData {
	_, err := c.Conn.Exec(c, `DELETE FROM education_article WHERE slug = $1`, c.PathParams["slug"])
	if err != nil {
		panic(err)
	}

	res := c.Redirect(hmnurl.BuildEducationIndex(), http.StatusSeeOther)
	res.AddFutureNotice("success", "Article deleted.")
	return res
}

func fetchEduArticles(
	ctx context.Context,
	dbConn db.ConnOrTx,
	t models.EduArticleType,
	currentUser *models.User,
) ([]models.EduArticle, error) {
	type eduArticleResult struct {
		Article        models.EduArticle        `db:"a"`
		CurrentVersion models.EduArticleVersion `db:"v"`
	}

	var qb db.QueryBuilder
	qb.Add(`
		SELECT $columns
		FROM
			education_article AS a
			JOIN education_article_version AS v ON a.current_version = v.id
		WHERE
			TRUE
	`)
	if t != 0 {
		qb.Add(`AND a.type = $?`, t)
	}
	if currentUser == nil || !currentUser.CanSeeUnpublishedEducationContent() {
		qb.Add(`AND NOT a.published`)
	}

	articles, err := db.Query[eduArticleResult](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return nil, err
	}

	var res []models.EduArticle
	for _, article := range articles {
		ver := article.CurrentVersion
		article.Article.CurrentVersion = &ver
		res = append(res, article.Article)
	}

	return res, nil
}

func fetchEduArticle(
	ctx context.Context,
	dbConn db.ConnOrTx,
	slug string,
	t models.EduArticleType,
	currentUser *models.User,
) (*models.EduArticle, error) {
	type eduArticleResult struct {
		Article        models.EduArticle        `db:"a"`
		CurrentVersion models.EduArticleVersion `db:"v"`
	}

	var qb db.QueryBuilder
	qb.Add(
		`
		SELECT $columns
		FROM
			education_article AS a
			JOIN education_article_version AS v ON a.current_version = v.id
		WHERE
			a.slug = $?
		`,
		slug,
	)
	if t != 0 {
		qb.Add(`AND a.type = $?`, t)
	}
	if currentUser == nil || !currentUser.CanSeeUnpublishedEducationContent() {
		qb.Add(`AND NOT a.published`)
	}

	res, err := db.QueryOne[eduArticleResult](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return nil, err
	}

	res.Article.CurrentVersion = &res.CurrentVersion
	return &res.Article, nil
}

func getEditorDataForEduArticle(
	urlContext *hmnurl.UrlContext,
	currentUser *models.User,
	baseData templates.BaseData,
	article *models.EduArticle,
) editorData {
	result := editorData{
		BaseData:    baseData,
		SubmitLabel: "Submit",

		CanEditPostTitle: true,
		MaxFileSize:      AssetMaxSize(currentUser),
		UploadUrl:        urlContext.BuildAssetUpload(),
		ShowEduOptions:   true,
		PreviewClass:     "edu-article",

		ParserName: "parseMarkdownEdu",
	}

	if article != nil {
		result.PostTitle = article.Title
		result.EditInitialContents = article.CurrentVersion.ContentRaw
	}

	return result
}

func getEduArticleFromForm(form url.Values) (art models.EduArticle, ver models.EduArticleVersion) {
	art.Title = form.Get("title")
	art.Slug = form.Get("slug")
	art.Description = form.Get("description")
	switch form.Get("type") {
	case "article":
		art.Type = models.EduArticleTypeArticle
	case "glossary":
		art.Type = models.EduArticleTypeGlossary
	default:
		panic(fmt.Errorf("unknown education article type: %s", form.Get("type")))
	}
	art.Published = form.Get("published") != ""

	ver.ContentRaw = form.Get("body")
	ver.ContentHTML = parsing.ParseMarkdown(ver.ContentRaw, parsing.EducationRealMarkdown)

	return
}

func createEduArticle(c *RequestContext, art models.EduArticle, ver models.EduArticleVersion) {
	tx := utils.Must1(c.Conn.Begin(c))
	defer tx.Rollback(c)
	{
		articleID := db.MustQueryOneScalar[int](c, tx,
			`
			INSERT INTO education_article (title, slug, description, published, type, current_version)
			VALUES                        ($1,    $2,   $3,          $4,        $5,   -1)
			RETURNING id
			`,
			art.Title, art.Slug, art.Description, art.Published, art.Type,
		)
		versionID := db.MustQueryOneScalar[int](c, tx,
			`
			INSERT INTO education_article_version (article_id, date, editor_id, content_raw, content_html)
			VALUES                                ($1,         $2,   $3,        $4,          $5          )
			RETURNING id
			`,
			articleID, time.Now(), c.CurrentUser.ID, ver.ContentRaw, ver.ContentHTML,
		)
		tx.Exec(c,
			`UPDATE education_article SET current_version = $1 WHERE id = $2`,
			versionID, articleID,
		)
	}
	utils.Must(tx.Commit(c))
}

func updateEduArticle(c *RequestContext, slug string, art models.EduArticle, ver models.EduArticleVersion) {
	tx := utils.Must1(c.Conn.Begin(c))
	defer tx.Rollback(c)
	{
		articleID := db.MustQueryOneScalar[int](c, tx,
			`SELECT id FROM education_article WHERE slug = $1`,
			slug,
		)
		versionID := db.MustQueryOneScalar[int](c, tx,
			`
			INSERT INTO education_article_version (article_id, date, editor_id, content_raw, content_html)
			VALUES                                ($1,         $2,   $3,        $4,          $5          )
			RETURNING id
			`,
			articleID, time.Now(), c.CurrentUser.ID, ver.ContentRaw, ver.ContentHTML,
		)
		tx.Exec(c,
			`
			UPDATE education_article
			SET
				title = $1, slug = $2, description = $3, published = $4, type = $5,
				current_version = $6
			WHERE
				id = $7
			`,
			art.Title, art.Slug, art.Description, art.Published, art.Type,
			versionID,
			articleID,
		)
	}
	utils.Must(tx.Commit(c))
}

func eduArticleURL(a *models.EduArticle) string {
	switch a.Type {
	case models.EduArticleTypeArticle:
		return hmnurl.BuildEducationArticle(a.Slug)
	case models.EduArticleTypeGlossary:
		return hmnurl.BuildEducationGlossary(a.Slug)
	default:
		panic("unknown education article type")
	}
}
