package website

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/utils"
)

type FormImage struct {
	New    bool
	Exists bool
	Remove bool

	// NOTE(ben): Will be set on New by reading the uploaded data.
	Filename string
	Mime     string
	Content  []byte
	Width    int
	Height   int
	Size     int64

	// NOTE(ben): Will be set if an existing asset ID was already present when
	// the page was loaded. May be present on Exists and Remove but not New.
	AssetID string
}

// NOTE(asaf): This assumes that you already called ParseMultipartForm (which
// is why there's no size limit here).
func GetFormImage(c *RequestContext, fieldName string) (FormImage, error) {
	// NOTE(ben): May be an asset UUID or "NOASSET".
	assetID := c.Req.Form.Get("original_" + fieldName)
	if assetID == "" {
		return FormImage{}, fmt.Errorf("field original_%s not found", fieldName)
	}

	img, header, err := c.Req.FormFile("image_" + fieldName)
	if errors.Is(err, http.ErrMissingFile) {
		removeAssetID := c.Req.Form.Get("remove_" + fieldName)
		if assetID == "NOASSET" {
			// No original file, and no file uploaded
			return FormImage{}, nil
		} else if removeAssetID != "" {
			// Existing file was removed

			// NOTE(ben): GetFormImage should only be used when there is one image
			// selector on the page with that name. This is basically an assert but
			// can fail on bad user input, so just an error here.
			if assetID != removeAssetID {
				return FormImage{}, errors.New("original and removed asset IDs do not match")
			}

			return FormImage{
				Remove:  true,
				AssetID: assetID,
			}, nil
		} else {
			// Existing image continues to exist
			return FormImage{
				Exists:  true,
				AssetID: assetID,
			}, nil
		}
	} else if err != nil {
		return FormImage{}, err
	}

	// NOTE(ben): In FormFile, if err == nil, header != nil.
	utils.Assert(header != nil)
	return loadFormImage(img, header)
}

// NOTE(ben): Roughly the same as GetFormImage, but gets N images that all
// share a name. Will also include images that should be removed.
func GetFormImages(c *RequestContext, fieldName string) ([]FormImage, error) {
	utils.Assert(c.Req.MultipartForm)

	// NOTE(ben): This whole thing is annoying because we need to synthesize a
	// list of all the images that belong under a particular name, whether they
	// were just uploaded, we are asking them to be removed, or they are existing
	// and unchanged, AND we need to have all of these in the same order they
	// appeared on the page.
	//
	// We do this by always having a nonzero field `original_<formname>` that
	// contains either a pre-existing asset ID or "NOASSET". If there are N
	// values of this field in the form, then there are N image selectors on the
	// page. We also have fields of `image_<formname>`, whose presence indicates
	// a newly-uploaded image, and `remove_<formname>`, whose presence indicates
	// an existing image that should be removed (and whose value is an asset ID
	// to be deleted).
	//
	// We can then iterate all values of `original_<formname>` and do the
	// following:
	//
	// - If there is any `remove_<formname>` such that `remove_<formname>` ==
	//   `original_<formname>`, that image should come back Remove.
	// - Otherwise, if `original_<formname>` is "NOASSET", iterate to the next
	//   value for `image_<formname>`; this is an uploaded image and should come
	//   back New.
	// - Otherwise, this image should come back Exists.
	//
	// In the end we should have iterated precisely all the values for
	// `image_<formname>`, so this can be a sanity check against bad form data.

	var res []FormImage
	var errs []error

	// NOTE(ben): FYI, you should read from c.Req.MultipartForm instead of
	// c.Req.Form for consistency with c.Req.MultipartForm.File and because .Form
	// can also be populated by query params, which we don't care about. (To be
	// fair, if people try to screw up our image loading using query params,
	// they're only hurting themselves.)

	assetIDsToRemove := make(map[string]struct{})
	for _, assetIDToRemove := range c.Req.MultipartForm.Value["remove_"+fieldName] {
		assetIDsToRemove[assetIDToRemove] = struct{}{}
	}

	nextNewImage := 0
	newImages := c.Req.MultipartForm.File["image_"+fieldName]
	for _, existingAssetID := range c.Req.MultipartForm.Value["original_"+fieldName] {
		if _, ok := assetIDsToRemove[existingAssetID]; ok {
			res = append(res, FormImage{
				Remove:  true,
				AssetID: existingAssetID,
			})
		} else if existingAssetID == "NOASSET" {
			if len(newImages) <= nextNewImage {
				errs = append(errs, errors.New("not enough new images"))
				continue
			}
			fileHeader := newImages[nextNewImage]
			nextNewImage++

			f, err := fileHeader.Open()
			if err != nil {
				errs = append(errs, err)
				continue
			}

			img, err := loadFormImage(f, fileHeader)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			res = append(res, img)
		} else {
			res = append(res, FormImage{
				Exists:  true,
				AssetID: existingAssetID,
			})
		}
	}
	if nextNewImage != len(c.Req.MultipartForm.File["image_"+fieldName]) {
		errs = append(errs, errors.New("incorrect number of new images"))
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return res, nil
}

func loadFormImage(file multipart.File, header *multipart.FileHeader) (FormImage, error) {
	res := FormImage{
		Filename: header.Filename,
		Content:  make([]byte, header.Size),
		Size:     header.Size,

		New: true,
	}

	file.Read(res.Content)
	file.Seek(0, io.SeekStart)

	fileExtensionOverrides := []string{".svg"}
	fileExt := strings.ToLower(path.Ext(res.Filename))
	tryDecode := true
	for _, ext := range fileExtensionOverrides {
		if fileExt == ext {
			tryDecode = false
		}
	}

	if tryDecode {
		config, _, err := image.DecodeConfig(file)
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

	return res, nil
}

func SaveFormImage(ctx context.Context, dbConn db.ConnOrTx, img FormImage, uploaderID *int) (*models.Asset, error) {
	utils.Assert(img.New)
	return assets.Create(ctx, dbConn, assets.CreateInput{
		Content:     img.Content,
		Filename:    img.Filename,
		ContentType: img.Mime,
		UploaderID:  uploaderID,
		Width:       img.Width,
		Height:      img.Height,
	})
}
