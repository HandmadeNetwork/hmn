package website

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/google/uuid"
	"github.com/tcolgate/mp3"
)

type PodcastIndexData struct {
	templates.BaseData
	Podcast       templates.Podcast
	Episodes      []templates.PodcastEpisode
	EditUrl       string
	NewEpisodeUrl string
}

func PodcastIndex(c *RequestContext) ResponseData {
	podcastResult, err := FetchPodcast(c, c.CurrentProject.ID, true, "")
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if podcastResult.Podcast == nil {
		return FourOhFour(c)
	}

	canEdit, err := CanEditProject(c, c.CurrentUser, podcastResult.Podcast.ProjectID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	baseData := getBaseDataAutocrumb(c, podcastResult.Podcast.Title)

	podcastIndexData := PodcastIndexData{
		BaseData: baseData,
		Podcast:  templates.PodcastToTemplate(c.CurrentProject.Slug, podcastResult.Podcast, podcastResult.ImageFile),
	}

	if canEdit {
		podcastIndexData.EditUrl = hmnurl.BuildPodcastEdit(c.CurrentProject.Slug)
		podcastIndexData.NewEpisodeUrl = hmnurl.BuildPodcastEpisodeNew(c.CurrentProject.Slug)
	}

	for _, episode := range podcastResult.Episodes {
		podcastIndexData.Episodes = append(podcastIndexData.Episodes, templates.PodcastEpisodeToTemplate(c.CurrentProject.Slug, episode, 0, podcastResult.ImageFile))
	}
	var res ResponseData
	err = res.WriteTemplate("podcast_index.html", podcastIndexData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render podcast index page"))
	}
	return res
}

type PodcastEditData struct {
	templates.BaseData
	Podcast templates.Podcast
}

func PodcastEdit(c *RequestContext) ResponseData {
	podcastResult, err := FetchPodcast(c, c.CurrentProject.ID, false, "")
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	canEdit, err := CanEditProject(c, c.CurrentUser, podcastResult.Podcast.ProjectID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if podcastResult.Podcast == nil || !canEdit {
		return FourOhFour(c)
	}

	podcast := templates.PodcastToTemplate(c.CurrentProject.Slug, podcastResult.Podcast, podcastResult.ImageFile)
	baseData := getBaseData(
		c,
		fmt.Sprintf("Edit %s", podcast.Title),
		[]templates.Breadcrumb{{Name: podcast.Title, Url: podcast.Url}},
	)
	podcastEditData := PodcastEditData{
		BaseData: baseData,
		Podcast:  podcast,
	}

	var res ResponseData
	err = res.WriteTemplate("podcast_edit.html", podcastEditData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render podcast edit page"))
	}
	return res
}

func PodcastEditSubmit(c *RequestContext) ResponseData {
	podcastResult, err := FetchPodcast(c, c.CurrentProject.ID, false, "")
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	canEdit, err := CanEditProject(c, c.CurrentUser, podcastResult.Podcast.ProjectID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if podcastResult.Podcast == nil || !canEdit {
		return FourOhFour(c)
	}

	c.Perf.StartBlock("PODCAST", "Handling file upload")
	c.Perf.StartBlock("PODCAST", "Parsing form")
	maxFileSize := int64(2 * 1024 * 1024)
	maxBodySize := maxFileSize + 1024*1024
	c.Req.Body = http.MaxBytesReader(c.Res, c.Req.Body, maxBodySize)
	err = c.Req.ParseMultipartForm(maxBodySize)
	c.Perf.EndBlock()
	if err != nil {
		// NOTE(asaf): The error for exceeding the max filesize doesn't have a special type, so we can't easily detect it here.
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse form"))
	}

	title := c.Req.Form.Get("title")
	if len(strings.TrimSpace(title)) == 0 {
		return RejectRequest(c, "Podcast title is empty")
	}
	description := c.Req.Form.Get("description")
	if len(strings.TrimSpace(description)) == 0 {
		return RejectRequest(c, "Podcast description is empty")
	}

	c.Perf.StartBlock("SQL", "Updating podcast")
	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to start db transaction"))
	}
	defer tx.Rollback(c.Context())

	imageSaveResult := SaveImageFile(c, tx, "podcast_image", maxFileSize, fmt.Sprintf("podcast/%s/logo%d", c.CurrentProject.Slug, time.Now().UTC().Unix()))
	if imageSaveResult.ValidationError != "" {
		return RejectRequest(c, imageSaveResult.ValidationError)
	} else if imageSaveResult.FatalError != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(imageSaveResult.FatalError, "Failed to save podcast image"))
	}

	if imageSaveResult.ImageFileID != 0 {
		_, err = tx.Exec(c.Context(),
			`
			UPDATE handmade_podcast
			SET
				title = $1,
				description = $2,
				image_id = $3
			WHERE id = $4
			`,
			title,
			description,
			imageSaveResult.ImageFileID,
			podcastResult.Podcast.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to update podcast"))
		}
	} else {
		_, err = tx.Exec(c.Context(),
			`
			UPDATE handmade_podcast
			SET
				title = $1,
				description = $2
			WHERE id = $3
			`,
			title,
			description,
			podcastResult.Podcast.ID,
		)
	}
	err = tx.Commit(c.Context())
	c.Perf.EndBlock()
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to commit db transaction"))
	}

	res := c.Redirect(hmnurl.BuildPodcastEdit(c.CurrentProject.Slug), http.StatusSeeOther)
	res.AddFutureNotice("success", "Podcast updated successfully.")
	return res
}

type PodcastEpisodeData struct {
	templates.BaseData
	Podcast templates.Podcast
	Episode templates.PodcastEpisode
	EditUrl string
}

func PodcastEpisode(c *RequestContext) ResponseData {
	episodeGUIDStr := c.PathParams["episodeid"]

	podcastResult, err := FetchPodcast(c, c.CurrentProject.ID, true, episodeGUIDStr)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if podcastResult.Podcast == nil || len(podcastResult.Episodes) == 0 {
		return FourOhFour(c)
	}

	canEdit, err := CanEditProject(c, c.CurrentUser, podcastResult.Podcast.ProjectID)
	if err != nil {
		c.Logger.Error().Err(err).Msg("Failed to check if user can edit podcast. Assuming they can't.") // NOTE(asaf): No need to return an error response here if it failed.
		canEdit = false
	}

	editUrl := ""
	if canEdit {
		editUrl = hmnurl.BuildPodcastEpisodeEdit(c.CurrentProject.Slug, podcastResult.Episodes[0].GUID.String())
	}

	podcast := templates.PodcastToTemplate(c.CurrentProject.Slug, podcastResult.Podcast, podcastResult.ImageFile)
	episode := templates.PodcastEpisodeToTemplate(c.CurrentProject.Slug, podcastResult.Episodes[0], 0, podcastResult.ImageFile)
	baseData := getBaseData(
		c,
		fmt.Sprintf("%s | %s", episode.Title, podcast.Title),
		[]templates.Breadcrumb{{Name: podcast.Title, Url: podcast.Url}},
	)

	podcastEpisodeData := PodcastEpisodeData{
		BaseData: baseData,
		Podcast:  podcast,
		Episode:  episode,
		EditUrl:  editUrl,
	}

	var res ResponseData
	err = res.WriteTemplate("podcast_episode.html", podcastEpisodeData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render podcast episode page"))
	}
	return res
}

type PodcastEpisodeEditData struct {
	templates.BaseData
	IsEdit        bool
	Title         string
	Description   string
	EpisodeNumber int
	CurrentFile   string
	EpisodeFiles  []string
}

func PodcastEpisodeNew(c *RequestContext) ResponseData {
	podcastResult, err := FetchPodcast(c, c.CurrentProject.ID, false, "")
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	canEdit, err := CanEditProject(c, c.CurrentUser, podcastResult.Podcast.ProjectID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if podcastResult.Podcast == nil || !canEdit {
		return FourOhFour(c)
	}

	episodeFiles, err := GetEpisodeFiles(c.CurrentProject.Slug)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to fetch podcast episode file list"))
	}

	podcast := templates.PodcastToTemplate(c.CurrentProject.Slug, podcastResult.Podcast, "")
	var res ResponseData
	baseData := getBaseData(
		c,
		fmt.Sprintf("New episode | %s", podcast.Title),
		[]templates.Breadcrumb{{Name: podcast.Title, Url: podcast.Url}},
	)
	err = res.WriteTemplate("podcast_episode_edit.html", PodcastEpisodeEditData{
		BaseData:     baseData,
		IsEdit:       false,
		EpisodeFiles: episodeFiles,
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render podcast episode new page"))
	}
	return res
}

func PodcastEpisodeEdit(c *RequestContext) ResponseData {
	episodeGUIDStr, found := c.PathParams["episodeid"]
	if !found || episodeGUIDStr == "" {
		return FourOhFour(c)
	}

	podcastResult, err := FetchPodcast(c, c.CurrentProject.ID, true, episodeGUIDStr)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	canEdit, err := CanEditProject(c, c.CurrentUser, podcastResult.Podcast.ProjectID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if podcastResult.Podcast == nil || len(podcastResult.Episodes) == 0 || !canEdit {
		return FourOhFour(c)
	}

	episodeFiles, err := GetEpisodeFiles(c.CurrentProject.Slug)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to fetch podcast episode file list"))
	}
	episode := podcastResult.Episodes[0]

	podcast := templates.PodcastToTemplate(c.CurrentProject.Slug, podcastResult.Podcast, "")
	podcastEpisode := templates.PodcastEpisodeToTemplate(c.CurrentProject.Slug, episode, 0, "")
	baseData := getBaseData(
		c,
		fmt.Sprintf("Edit episode %s | %s", podcastEpisode.Title, podcast.Title),
		[]templates.Breadcrumb{{Name: podcast.Title, Url: podcast.Url}, {Name: podcastEpisode.Title, Url: podcastEpisode.Url}},
	)
	podcastEpisodeEditData := PodcastEpisodeEditData{
		BaseData:      baseData,
		IsEdit:        true,
		Title:         episode.Title,
		Description:   episode.Description,
		EpisodeNumber: episode.EpisodeNumber,
		CurrentFile:   episode.AudioFile,
		EpisodeFiles:  episodeFiles,
	}

	var res ResponseData
	err = res.WriteTemplate("podcast_episode_edit.html", podcastEpisodeEditData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render podcast episode edit page"))
	}
	return res
}

func PodcastEpisodeSubmit(c *RequestContext) ResponseData {
	episodeGUIDStr, found := c.PathParams["episodeid"]

	isEdit := found && episodeGUIDStr != ""
	podcastResult, err := FetchPodcast(c, c.CurrentProject.ID, isEdit, episodeGUIDStr)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	canEdit, err := CanEditProject(c, c.CurrentUser, podcastResult.Podcast.ProjectID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if podcastResult.Podcast == nil || (isEdit && len(podcastResult.Episodes) == 0) || !canEdit {
		return FourOhFour(c)
	}

	c.Perf.StartBlock("OS", "Fetching podcast episode files")
	episodeFiles, err := GetEpisodeFiles(c.CurrentProject.Slug)
	c.Perf.EndBlock()
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to fetch podcast episode file list"))
	}

	c.Req.ParseForm()
	title := c.Req.Form.Get("title")
	if len(strings.TrimSpace(title)) == 0 {
		return RejectRequest(c, "Episode title is empty")
	}
	description := c.Req.Form.Get("description")
	if len(strings.TrimSpace(description)) == 0 {
		return RejectRequest(c, "Episode description is empty")
	}
	episodeNumberStr := c.Req.Form.Get("episode_number")
	episodeNumber, err := strconv.Atoi(episodeNumberStr)
	if err != nil {
		return RejectRequest(c, "Episode number can't be parsed")
	}
	episodeFile := c.Req.Form.Get("episode_file")
	found = false
	for _, ef := range episodeFiles {
		if episodeFile == ef {
			found = true
			break
		}
	}

	if !found {
		return RejectRequest(c, "Requested episode file not found")
	}

	c.Perf.StartBlock("MP3", "Parsing mp3 file for duration")
	file, err := os.Open(fmt.Sprintf("public/media/podcast/%s/%s", c.CurrentProject.Slug, episodeFile))
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to open podcast file"))
	}

	mp3Decoder := mp3.NewDecoder(file)
	var duration float64
	skipped := 0
	var decodingError error
	var f mp3.Frame
	for {
		if err = mp3Decoder.Decode(&f, &skipped); err != nil {
			if err == io.EOF {
				break
			} else {
				decodingError = err
				break
			}
		}
		duration = duration + f.Duration().Seconds()
	}
	file.Close()
	c.Perf.EndBlock()
	if decodingError != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to decode mp3 file"))
	}

	c.Perf.StartBlock("MARKDOWN", "Parsing description")
	descriptionRendered := parsing.ParseMarkdown(description, parsing.ForumRealMarkdown)
	c.Perf.EndBlock()

	guidStr := ""
	if isEdit {
		guidStr = podcastResult.Episodes[0].GUID.String()
		c.Perf.StartBlock("SQL", "Updating podcast episode")
		_, err := c.Conn.Exec(c.Context(),
			`
			UPDATE handmade_podcastepisode
			SET
				title = $1,
				description = $2,
				description_rendered = $3,
				audio_filename = $4,
				duration = $5,
				episode_number = $6
			WHERE
				guid = $7
			`,
			title,
			description,
			descriptionRendered,
			episodeFile,
			duration,
			episodeNumber,
			podcastResult.Episodes[0].GUID,
		)
		c.Perf.EndBlock()
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to update podcast episode"))
		}
	} else {
		guid := uuid.New()
		guidStr = guid.String()
		c.Perf.StartBlock("SQL", "Creating new podcast episode")
		_, err := c.Conn.Exec(c.Context(),
			`
			INSERT INTO handmade_podcastepisode
				(guid, title, description, description_rendered, audio_filename, duration, pub_date, episode_number, podcast_id)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`,
			guid,
			title,
			description,
			descriptionRendered,
			episodeFile,
			duration,
			time.Now(),
			episodeNumber,
			podcastResult.Podcast.ID,
		)
		c.Perf.EndBlock()
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to create podcast episode"))
		}
	}

	res := c.Redirect(hmnurl.BuildPodcastEpisodeEdit(c.CurrentProject.Slug, guidStr), http.StatusSeeOther)
	res.AddFutureNotice("success", "Podcast episode updated successfully.")
	return res
}

func GetEpisodeFiles(projectSlug string) ([]string, error) {
	folderStr := fmt.Sprintf("public/media/podcast/%s/", projectSlug)
	folder := os.DirFS(folderStr)
	files, err := fs.Glob(folder, "*.mp3")
	return files, err
}

type PodcastRSSData struct {
	Podcast  templates.Podcast
	Episodes []templates.PodcastEpisode
}

func PodcastRSS(c *RequestContext) ResponseData {
	podcastResult, err := FetchPodcast(c, c.CurrentProject.ID, true, "")
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if podcastResult.Podcast == nil {
		return FourOhFour(c)
	}

	podcastRSSData := PodcastRSSData{
		Podcast: templates.PodcastToTemplate(c.CurrentProject.Slug, podcastResult.Podcast, podcastResult.ImageFile),
	}

	for _, episode := range podcastResult.Episodes {
		var filesize int64
		stat, err := os.Stat(fmt.Sprintf("./public/media/podcast/%s/%s", c.CurrentProject.Slug, episode.AudioFile))
		if err != nil {
			c.Logger.Err(err).Msg("Couldn't get filesize for podcast episode")
		} else {
			filesize = stat.Size()
		}
		podcastRSSData.Episodes = append(podcastRSSData.Episodes, templates.PodcastEpisodeToTemplate(c.CurrentProject.Slug, episode, filesize, podcastResult.ImageFile))
	}

	var res ResponseData
	err = res.WriteTemplate("podcast.xml", podcastRSSData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render podcast RSS"))
	}
	return res
}

type PodcastResult struct {
	Podcast   *models.Podcast
	ImageFile string
	Episodes  []*models.PodcastEpisode
}

func FetchPodcast(c *RequestContext, projectId int, fetchEpisodes bool, episodeGUID string) (PodcastResult, error) {
	var result PodcastResult
	c.Perf.StartBlock("SQL", "Fetch podcast")
	type podcastQuery struct {
		Podcast       models.Podcast `db:"podcast"`
		ImageFilename string         `db:"imagefile.file"`
	}
	podcastQueryResult, err := db.QueryOne(c.Context(), c.Conn, podcastQuery{},
		`
		SELECT $columns
		FROM handmade_podcast AS podcast
		LEFT JOIN handmade_imagefile AS imagefile ON imagefile.id = podcast.image_id
		WHERE podcast.project_id = $1
		`,
		projectId,
	)
	c.Perf.EndBlock()
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return result, nil
		} else {
			return result, oops.New(err, "failed to fetch podcast")
		}
	}
	podcast := podcastQueryResult.(*podcastQuery).Podcast
	podcastImageFilename := podcastQueryResult.(*podcastQuery).ImageFilename
	result.Podcast = &podcast
	result.ImageFile = podcastImageFilename

	if fetchEpisodes {
		type podcastEpisodeQuery struct {
			Episode models.PodcastEpisode `db:"episode"`
		}
		if episodeGUID == "" {
			c.Perf.StartBlock("SQL", "Fetch podcast episodes")
			podcastEpisodeQueryResult, err := db.Query(c.Context(), c.Conn, podcastEpisodeQuery{},
				`
				SELECT $columns
				FROM handmade_podcastepisode AS episode
				WHERE episode.podcast_id = $1
				ORDER BY episode.season_number DESC, episode.episode_number DESC
				`,
				podcast.ID,
			)
			c.Perf.EndBlock()
			if err != nil {
				return result, oops.New(err, "failed to fetch podcast episodes")
			}
			for _, episodeRow := range podcastEpisodeQueryResult.ToSlice() {
				result.Episodes = append(result.Episodes, &episodeRow.(*podcastEpisodeQuery).Episode)
			}
		} else {
			guid, err := uuid.Parse(episodeGUID)
			if err != nil {
				return result, err
			}
			c.Perf.StartBlock("SQL", "Fetch podcast episode")
			podcastEpisodeQueryResult, err := db.QueryOne(c.Context(), c.Conn, podcastEpisodeQuery{},
				`
				SELECT $columns
				FROM handmade_podcastepisode AS episode
				WHERE episode.podcast_id = $1 AND episode.guid = $2
				`,
				podcast.ID,
				guid,
			)
			c.Perf.EndBlock()
			if err != nil {
				if errors.Is(err, db.ErrNoMatchingRows) {
					return result, nil
				} else {
					return result, oops.New(err, "failed to fetch podcast episode")
				}
			}
			episode := podcastEpisodeQueryResult.(*podcastEpisodeQuery).Episode
			result.Episodes = append(result.Episodes, &episode)
		}
	}

	return result, nil
}
