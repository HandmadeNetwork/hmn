package website

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
)

type AssetUploadResult struct {
	Url   string `json:"url,omitempty"`
	Mime  string `json:"mime,omitempty"`
	Error string `json:"error,omitempty"`
}

const assetMaxSize = 10 * 1024 * 1024
const assetMaxSizeAdmin = 10 * 1024 * 1024 * 1024

func AssetMaxSize(user *models.User) int {
	if user != nil && user.IsStaff {
		return assetMaxSizeAdmin
	} else {
		return assetMaxSize
	}
}

func AssetUpload(c *RequestContext) ResponseData {
	maxFilesize := AssetMaxSize(c.CurrentUser)

	contentLength, hasLength := c.Req.Header["Content-Length"]
	if hasLength {
		filesize, err := strconv.Atoi(contentLength[0])
		if err == nil && filesize > maxFilesize {
			res := ResponseData{
				StatusCode: http.StatusOK,
			}
			jsonString, _ := json.Marshal(AssetUploadResult{
				Error: fmt.Sprintf("Filesize too big. Maximum size is %d.", maxFilesize),
			})
			res.Write(jsonString)
			return res
		}
	}

	filenameHeader, hasFilename := c.Req.Header["Hmn-Upload-Filename"]
	originalFilename := "upload"
	if hasFilename {
		decodedFilename, err := base64.StdEncoding.DecodeString(filenameHeader[0])
		if err == nil {
			originalFilename = string(decodedFilename)
		}
	}

	bodyReader := http.MaxBytesReader(c.Res, c.Req.Body, int64(maxFilesize))
	data, err := io.ReadAll(bodyReader)
	if err != nil {
		res := ResponseData{
			StatusCode: http.StatusBadRequest,
			Errors:     []error{err},
		}
		return res
	}

	mimeType := http.DetectContentType(data)
	width := 0
	height := 0

	if strings.HasPrefix(mimeType, "image") {
		config, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err == nil {
			width = config.Width
			height = config.Height
		} else {
			// NOTE(asaf): Not image
			mimeType = "application/octet-stream"
		}
	}

	asset, err := assets.Create(c, c.Conn, assets.CreateInput{
		Content:     data,
		Filename:    originalFilename,
		ContentType: mimeType,
		UploaderID:  &c.CurrentUser.ID,
		Width:       width,
		Height:      height,
	})

	if err != nil {
		res := ResponseData{
			StatusCode: http.StatusBadRequest,
			Errors:     []error{err},
		}
		return res
	}

	res := ResponseData{
		StatusCode: http.StatusOK,
	}
	jsonString, err := json.Marshal(AssetUploadResult{
		Url:  hmnurl.BuildS3Asset(asset.S3Key),
		Mime: asset.MimeType,
	})
	res.Write(jsonString)
	return res
}
