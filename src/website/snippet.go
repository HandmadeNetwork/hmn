package website

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/embed"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/google/uuid"
	"mvdan.cc/xurls/v2"
)

type SnippetData struct {
	templates.BaseData
	Snippet templates.TimelineItem

	CanEditSnippet bool
	SnippetEdit    templates.SnippetEdit
}

func Snippet(c *RequestContext) ResponseData {
	snippetId := -1
	snippetIdStr, found := c.PathParams["snippetid"]
	if found && snippetIdStr != "" {
		var err error
		if snippetId, err = strconv.Atoi(snippetIdStr); err != nil {
			return FourOhFour(c)
		}
	}
	if snippetId < 1 {
		return FourOhFour(c)
	}

	s, err := hmndata.FetchSnippet(c, c.Conn, c.CurrentUser, snippetId, hmndata.SnippetQuery{})
	if err != nil {
		if errors.Is(err, db.NotFound) {
			return FourOhFour(c)
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippet"))
		}
	}

	canEdit := (c.CurrentUser != nil && (c.CurrentUser.IsStaff || c.CurrentUser.ID == s.Owner.ID))
	snippet := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, canEdit)

	opengraph := []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "article"},
		{Property: "og:url", Value: snippet.Url},
		{Property: "og:title", Value: fmt.Sprintf("Snippet by %s", snippet.OwnerName)},
		{Property: "og:description", Value: string(snippet.Description)},
	}

	if len(snippet.Media) > 0 {
		media := snippet.Media[0]

		switch media.Type {
		case templates.TimelineItemMediaTypeImage:
			opengraph = append(opengraph,
				templates.OpenGraphItem{Property: "og:image", Value: media.AssetUrl},
				templates.OpenGraphItem{Property: "og:image:width", Value: strconv.Itoa(media.Width)},
				templates.OpenGraphItem{Property: "og:image:height", Value: strconv.Itoa(media.Height)},
				templates.OpenGraphItem{Property: "og:image:type", Value: media.MimeType},
				templates.OpenGraphItem{Name: "twitter:card", Value: "summary_large_image"},
			)
		case templates.TimelineItemMediaTypeVideo:
			opengraph = append(opengraph,
				templates.OpenGraphItem{Property: "og:video", Value: media.AssetUrl},
				templates.OpenGraphItem{Property: "og:video:width", Value: strconv.Itoa(media.Width)},
				templates.OpenGraphItem{Property: "og:video:height", Value: strconv.Itoa(media.Height)},
				templates.OpenGraphItem{Property: "og:video:type", Value: media.MimeType},
				templates.OpenGraphItem{Name: "twitter:card", Value: "player"},
			)
		case templates.TimelineItemMediaTypeAudio:
			opengraph = append(opengraph,
				templates.OpenGraphItem{Property: "og:audio", Value: media.AssetUrl},
				templates.OpenGraphItem{Property: "og:audio:type", Value: media.MimeType},
				templates.OpenGraphItem{Name: "twitter:card", Value: "player"},
			)
		}
		opengraph = append(opengraph, media.ExtraOpenGraphItems...)
	}

	baseData := getBaseData(c, fmt.Sprintf("Snippet by %s", snippet.OwnerName))
	baseData.OpenGraphItems = opengraph // NOTE(asaf): We're overriding the defaults on purpose.
	snippetEdit := templates.SnippetEdit{}
	if c.CurrentUser != nil {
		userProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user projects"))
		}
		templateProjects := make([]templates.Project, 0, len(userProjects))
		for _, p := range userProjects {
			templateProject := templates.ProjectAndStuffToTemplate(&p)
			templateProjects = append(templateProjects, templateProject)
		}
		snippetEdit = templates.SnippetEdit{
			AvailableProjectsJSON: templates.SnippetEditProjectsToJSON(templateProjects),
			SubmitUrl:             hmnurl.BuildSnippetSubmit(),
			AssetMaxSize:          AssetMaxSize(c.CurrentUser),
		}
	}
	var res ResponseData
	err = res.WriteTemplate("snippet.html", SnippetData{
		BaseData:       baseData,
		Snippet:        snippet,
		CanEditSnippet: canEdit,
		SnippetEdit:    snippetEdit,
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render snippet template"))
	}
	return res
}

func SnippetEditSubmit(c *RequestContext) ResponseData {
	maxUploadSize := AssetMaxSize(c.CurrentUser)
	maxBodySize := int64(maxUploadSize + 1024*1024)
	c.Req.Body = http.MaxBytesReader(c.Res, c.Req.Body, maxBodySize)
	err := c.Req.ParseMultipartForm(maxBodySize)
	if err != nil {
		// NOTE(asaf): The error for exceeding the max filesize doesn't have a special type, so we can't easily detect it here.
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse form"))
	}

	form := c.Req.PostForm

	redirect := form.Get("redirect")
	action := form.Get("action")
	existingSnippetIdStr := strings.TrimSpace(form.Get("snippet_id"))
	var existingSnippet *hmndata.SnippetAndStuff
	originalText := ""
	var embedUrl *string
	var assetID *uuid.UUID

	if len(existingSnippetIdStr) > 0 {
		existingSnippetId, err := strconv.Atoi(existingSnippetIdStr)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse snippet id"))
		}
		query := hmndata.SnippetQuery{}
		if !c.CurrentUser.IsStaff {
			query.OwnerIDs = []int{c.CurrentUser.ID}
		}
		snip, err := hmndata.FetchSnippet(c, c.Conn, c.CurrentUser, existingSnippetId, query)
		if err != nil {
			if errors.Is(err, db.NotFound) {
				return FourOhFour(c)
			} else {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch existing snippet for edit"))
			}
		}
		originalText = snip.Snippet.Description
		embedUrl = snip.Snippet.Url
		assetID = snip.Snippet.AssetID
		if snip.Snippet.Url != nil {
			embedUrl = snip.Snippet.Url
		}
		existingSnippet = &snip
	}

	if strings.ToLower(action) == "delete" {
		if existingSnippet != nil {
			_, err = c.Conn.Exec(c,
				`
				DELETE FROM snippet
				WHERE snippet.id = $1
				`,
				existingSnippet.Snippet.ID,
			)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch existing snippet for edit"))
			}
		} else {
			return FourOhFour(c)
		}
	} else {
		if form.Get("remove_attachment") == "true" {
			embedUrl = nil
			assetID = nil
		}
		text := strings.TrimSpace(form.Get("text"))
		textHtml := parsing.ParseMarkdown(text, parsing.DiscordMarkdown)
		projectAssociations := form["project_id"]
		var assetData *assets.CreateInput

		file, header, err := c.Req.FormFile("file")
		if err != nil && !errors.Is(err, http.ErrMissingFile) {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get form file"))
		}

		if header != nil {
			content := make([]byte, header.Size)
			_, err = file.Read(content)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to read uploaded file"))
			}
			contentType := header.Header.Get("Content-Type")
			if contentType == "" {
				contentType = http.DetectContentType(content)
			}
			width := 0
			height := 0
			if strings.HasPrefix(contentType, "image/") && contentType != "image/svg+xml" {
				file.Seek(0, io.SeekStart)
				config, _, err := image.DecodeConfig(file)
				if err == nil {
					width = config.Width
					height = config.Height
				}
			}
			assetData = &assets.CreateInput{
				Content:     content,
				Filename:    header.Filename,
				ContentType: contentType,
				UploaderID:  &c.CurrentUser.ID,
				Width:       width,
				Height:      height,
			}
		}

		if originalText != text && assetData == nil && embedUrl == nil && assetID == nil {
			urls := xurls.Relaxed().FindAllString(text, -1)
			if urls != nil {
				embeddable, err := embed.GetEmbeddableFromUrls(c, urls, maxUploadSize, time.Second*10, 3)
				if err != nil {
					if !errors.Is(err, embed.DownloadTooBigError) && !errors.Is(err, embed.NoEmbedFound) {
						c.Logger.Error().Err(err).Msg("failed to fetch embeddable for snippet")
					}
				} else {
					if embeddable.Url != "" {
						embedUrl = &embeddable.Url
					} else {
						width := 0
						height := 0
						if strings.HasPrefix(embeddable.File.ContentType, "image/") && embeddable.File.ContentType != "image/svg+xml" {
							reader := bytes.NewReader(embeddable.File.Data)
							config, _, err := image.DecodeConfig(reader)
							if err == nil {
								width = config.Width
								height = config.Height
							}
						}
						assetData = &assets.CreateInput{
							Content:     embeddable.File.Data,
							Filename:    embeddable.File.Filename,
							ContentType: embeddable.File.ContentType,
							UploaderID:  &c.CurrentUser.ID,
							Width:       width,
							Height:      height,
						}
					}
				}
			}
		}

		if text == "" && assetData == nil && embedUrl == nil && assetID == nil {
			return c.RejectRequest("You must provide a description or a file attachment.")
		}

		tx, err := c.Conn.Begin(c)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to start transaction"))
		}
		defer tx.Rollback(c)

		var asset *models.Asset
		if assetData != nil {
			asset, err = assets.Create(c, tx, *assetData)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create asset"))
			}
			assetID = &asset.ID
		}

		snippetId := 0
		if existingSnippet != nil {
			_, err = tx.Exec(c,
				`
				UPDATE snippet SET 
					url = $2,
					description = $3,
					_description_html = $4,
					asset_id = $5,
					edited_on_website = $6
				WHERE id = $1
				`,
				existingSnippet.Snippet.ID,
				embedUrl,
				text,
				textHtml,
				assetID,
				true,
			)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update snippet"))
			}
			snippetId = existingSnippet.Snippet.ID
		} else {
			newSnippetId, err := db.QueryOne[int](c, tx,
				`
				INSERT INTO snippet (url, "when", description, _description_html, asset_id, owner_id, edited_on_website)
				VALUES ($1, $2, $3, $4, $5, $6 ,$7)
				RETURNING id
				`,
				embedUrl,
				time.Now(),
				text,
				textHtml,
				assetID,
				c.CurrentUser.ID,
				true,
			)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to insert snippet"))
			}
			snippetId = *newSnippetId
		}

		_, err = tx.Exec(c,
			`
			DELETE FROM snippet_project
			WHERE snippet_id = $1
			`,
			snippetId,
		)

		if len(projectAssociations) > 0 {
			var projectIds []int
			for _, pidStr := range projectAssociations {
				projId, err := strconv.Atoi(pidStr)
				if err != nil {
					continue
				}
				projectIds = append(projectIds, projId)
			}

			if len(projectIds) > 0 {
				projectQuery := hmndata.ProjectsQuery{
					ProjectIDs: projectIds,
				}
				if !c.CurrentUser.IsStaff {
					projectQuery.OwnerIDs = []int{c.CurrentUser.ID}
				}
				projects, err := hmndata.FetchProjects(c, tx, c.CurrentUser, projectQuery)
				if err != nil {
					return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch projects for snippet"))
				}
				for _, p := range projects {
					_, err = tx.Exec(c,
						`
						INSERT INTO snippet_project (snippet_id, project_id, kind)
						VALUES ($1, $2, $3)
						`,
						snippetId,
						p.Project.ID,
						models.SnippetProjectKindWebsite,
					)
				}
			}
		}

		hmndata.UpdateSnippetLastPostedForAllProjects(c, tx)

		err = tx.Commit(c)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit transaction"))
		}
	}

	return c.Redirect(redirect, http.StatusSeeOther)
}
