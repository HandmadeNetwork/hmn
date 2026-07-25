package website

import (
	"context"
	"errors"
	"image"
	"io"
	"net/http"
	"path"
	"strings"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/utils"
)

type FormImage struct {
	Exists   bool
	Remove   bool
	Filename string
	Mime     string
	Content  []byte
	Width    int
	Height   int
	Size     int64
}

// NOTE(asaf): This assumes that you already called ParseMultipartForm (which is why there's no size limit here).
func GetFormImage(c *RequestContext, fieldName string) (FormImage, error) {
	var res FormImage
	res.Exists = false

	removeStr := c.Req.Form.Get("remove_" + fieldName)
	res.Remove = (removeStr == "true")
	img, header, err := c.Req.FormFile(fieldName)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return res, nil
		} else {
			return FormImage{}, err
		}
	}

	if header != nil {
		res.Exists = true
		res.Size = header.Size
		res.Filename = header.Filename

		res.Content = make([]byte, res.Size)
		img.Read(res.Content)
		img.Seek(0, io.SeekStart)

		fileExtensionOverrides := []string{".svg"}
		fileExt := strings.ToLower(path.Ext(res.Filename))
		tryDecode := true
		for _, ext := range fileExtensionOverrides {
			if fileExt == ext {
				tryDecode = false
			}
		}

		if tryDecode {
			config, _, err := image.DecodeConfig(img)
			if err != nil {
				return FormImage{}, err
			}
			res.Width = config.Width
			res.Height = config.Height
			res.Mime = http.DetectContentType(res.Content)
		} else {
			if fileExt == ".svg" {
				res.Mime = "image/svg+xml"
			}
		}
	}

	return res, nil
}

func SaveFormImage(ctx context.Context, dbConn db.ConnOrTx, img FormImage, uploaderID *int) (*models.Asset, error) {
	utils.Assert(img.Exists)
	return assets.Create(ctx, dbConn, assets.CreateInput{
		Content:     img.Content,
		Filename:    img.Filename,
		ContentType: img.Mime,
		UploaderID:  uploaderID,
		Width:       img.Width,
		Height:      img.Height,
	})
}
