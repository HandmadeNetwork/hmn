package initimage

import (
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp" // NOTE(asaf): webp handles vp8 and vp8l
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)
