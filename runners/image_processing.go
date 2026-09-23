package runners

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/jdeng/goheif"
	"github.com/rwcarlsen/goexif/exif"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// decodeImage decodes the image at filePath, applying EXIF orientation for
// formats that carry it (JPEG/TIFF) and handling HEIC/HEIF specially.
func decodeImage(file *os.File, filePath string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".heic", ".heif":
		return decodeHEIC(file)
	default:
		return correctImageOrientation(file)
	}
}

// decodeHEIC decodes a HEIC/HEIF image and applies any EXIF orientation.
func decodeHEIC(file *os.File) (image.Image, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	img, err := goheif.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode HEIC: %w", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	exifData, err := goheif.ExtractExif(file)
	if err != nil || exifData == nil {
		return img, nil
	}

	x, err := exif.Decode(bytes.NewReader(exifData))
	if err != nil || x == nil {
		return img, nil
	}

	orientTag, err := x.Get(exif.Orientation)
	if err != nil {
		return img, nil
	}

	orientation, err := orientTag.Int(0)
	if err != nil {
		return img, nil
	}

	return applyOrientation(img, orientation), nil
}

// correctImageOrientation decodes a standard image (JPEG/TIFF/etc.) and applies
// any EXIF orientation.
func correctImageOrientation(file *os.File) (image.Image, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	x, exifErr := exif.Decode(file)

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	if exifErr != nil || x == nil {
		return img, nil
	}

	orient, err := x.Get(exif.Orientation)
	if err != nil {
		return img, nil
	}
	orientation, err := orient.Int(0)
	if err != nil {
		return img, nil
	}

	return applyOrientation(img, orientation), nil
}

// applyOrientation rotates/flips img according to an EXIF orientation value.
func applyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.Transpose(img)
	case 6:
		return imaging.Rotate270(img)
	case 7:
		return imaging.Transverse(img)
	case 8:
		return imaging.Rotate90(img)
	default:
		return img
	}
}
