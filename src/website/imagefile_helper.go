package website

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/oops"
)

// If a helper method returns this, you should call RejectRequest with the value.
type RejectRequestError error

/*
Reads an image file from form data and saves it to the filesystem and the database.
If the file doesn't exist, this does nothing.

NOTE(ben): Someday we should replace this with the asset system.
*/
func SaveImageFile(c *RequestContext, dbConn db.ConnOrTx, fileFieldName string, maxSize int64, filepath string) (imageFileId int, err error) {
	img, header, err := c.Req.FormFile(fileFieldName)
	filename := ""
	width := 0
	height := 0
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		return 0, oops.New(err, "failed to read uploaded file")
	}

	if header != nil {
		if header.Size > maxSize {
			return 0, RejectRequestError(fmt.Errorf("Image filesize too big. Max size: %d bytes", maxSize))
		} else {
			c.Perf.StartBlock("IMAGE", "Decoding image")
			config, format, err := image.DecodeConfig(img)
			c.Perf.EndBlock()
			if err != nil {
				return 0, RejectRequestError(errors.New("Image type not supported"))
			}
			width = config.Width
			height = config.Height
			if width == 0 || height == 0 {
				return 0, RejectRequestError(errors.New("Image has zero size"))
			}

			filename = fmt.Sprintf("%s.%s", filepath, format)
			storageFilename := fmt.Sprintf("public/media/%s", filename)
			c.Perf.StartBlock("IMAGE", "Writing image file")
			file, err := os.Create(storageFilename)
			if err != nil {
				return 0, oops.New(err, "Failed to create local image file")
			}
			img.Seek(0, io.SeekStart)
			_, err = io.Copy(file, img)
			if err != nil {
				return 0, oops.New(err, "Failed to write image to file")
			}
			file.Close()
			img.Close()
			c.Perf.EndBlock()
		}
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Saving image file")
	if filename != "" {
		hasher := sha1.New()
		img.Seek(0, io.SeekStart)
		io.Copy(hasher, img) // NOTE(asaf): Writing to hash.Hash never returns an error according to the docs
		sha1sum := hasher.Sum(nil)
		var imageId int
		err = dbConn.QueryRow(c.Context(),
			`
			INSERT INTO handmade_imagefile (file, size, sha1sum, protected, width, height)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
			`,
			filename, header.Size, hex.EncodeToString(sha1sum), false, width, height,
		).Scan(&imageId)
		if err != nil {
			return 0, oops.New(err, "Failed to insert image file row")
		}

		return imageId, nil
	}

	return 0, nil
}
