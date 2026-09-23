package runners

import (
	"angadrive/database"
	"angadrive/globals"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

// ImageJob is a request to convert an image file to a target format.
type ImageJob struct {
	File database.FileData
	// TargetFormat selects the output format ("png" or "jpg"). When empty, the
	// runner infers the target from the source extension (PNG source -> JPEG,
	// otherwise -> PNG) for backward compatibility.
	TargetFormat string
}

// ID returns the dedup key for the job (the file's content hash).
func (j ImageJob) ID() string {
	return j.File.Sha256sum
}

// ImageRunner converts non-PNG image files to lossless PNG.
//
// It is a concrete Runner built on the generic runners framework. Decoding and
// re-encoding a full-resolution image is CPU-intensive, so it is queued and run
// with bounded concurrency by the Manager.
type ImageRunner struct {
	notifier Notifier
}

// Name returns the unique identifier used to submit image jobs.
func (r *ImageRunner) Name() string {
	return "image"
}

// Run converts a single image file. It is called from a worker goroutine.
func (r *ImageRunner) Run(job Job) error {
	ij, ok := job.(ImageJob)
	if !ok {
		return fmt.Errorf("image runner received unexpected job type %T", job)
	}
	return r.convert(ij.File, ij.TargetFormat)
}

// convert decodes the source image and re-encodes it in the target format
// (PNG source -> JPEG, otherwise -> lossless PNG), registering the result as a
// new file and notifying the user.
func (r *ImageRunner) convert(inputFile database.FileData, targetFormat string) error {
	inputFilePath := globals.UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + inputFile.Sha256sum
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "input file not found: " + inputFile.OriginalFileName,
			},
		})
		return fmt.Errorf("input file not found: %s", inputFile.OriginalFileName)
	}

	file, err := os.Open(inputFilePath)
	if err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to open input file: " + err.Error(),
			},
		})
		return err
	}
	defer file.Close()

	img, err := decodeImage(file, inputFilePath)
	if err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to decode image: " + err.Error(),
			},
		})
		return err
	}

	// Determine conversion direction from the requested target format. When no
	// target is given, fall back to the source extension (PNG sources are
	// converted to a small JPEG; every other image format is converted to a
	// losslessly compressed PNG).
	srcExt := strings.ToLower(filepath.Ext(inputFilePath))
	isPngSource := srcExt == ".png"

	target := strings.ToLower(strings.TrimSpace(targetFormat))
	if target == "" {
		if isPngSource {
			target = "jpg"
		} else {
			target = "png"
		}
	}
	if target != "png" && target != "jpg" {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "unsupported target format: " + targetFormat,
			},
		})
		return fmt.Errorf("unsupported target format: %s", targetFormat)
	}

	tempSuffix := ".png-convert-*.tmp"
	outExt := ".png"
	outFormat := imaging.PNG
	encodeOpts := imaging.PNGCompressionLevel(png.BestCompression)
	if target == "jpg" {
		tempSuffix = ".jpg-convert-*.tmp"
		outExt = ".jpg"
		outFormat = imaging.JPEG
		// Quality 65 keeps the file small while reasonably preserving quality.
		encodeOpts = imaging.JPEGQuality(65)
	}

	// Write to a temporary file so a partially-written output is never exposed.
	tempFile, err := os.CreateTemp(globals.UPLOAD_DIR+string(os.PathSeparator)+"i", tempSuffix)
	if err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to create temporary output: " + err.Error(),
			},
		})
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	// Direction-aware encoding: PNG source -> JPEG (quality 65), else -> PNG
	// lossless with maximum compression.
	if err := imaging.Encode(tempFile, img, outFormat, encodeOpts); err != nil {
		tempFile.Close()
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to encode image: " + err.Error(),
			},
		})
		return err
	}
	if err := tempFile.Close(); err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to close temporary output: " + err.Error(),
			},
		})
		return err
	}

	fileInfo, err := os.Stat(tempPath)
	if err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to get output file stats: " + err.Error(),
			},
		})
		return err
	}
	fileSize := fileInfo.Size()

	outputSha256sum, err := sha256sum(tempPath)
	if err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to calculate SHA-256 checksum: " + err.Error(),
			},
		})
		return err
	}

	outputFilePath := globals.UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + outputSha256sum + outExt
	if err := os.Rename(tempPath, outputFilePath); err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to rename output file: " + err.Error(),
			},
		})
		return err
	}

	uniqueFileName := database.GenerateUniqueFileName(removeExtension(inputFile.OriginalFileName) + outExt)
	fileData := database.FileData{
		OriginalFileName: removeExtension(inputFile.OriginalFileName) + outExt,
		FileDirectory:    uniqueFileName,
		AccountToken:     inputFile.AccountToken,
		FileSize:         fileSize,
		Timestamp:        time.Now().UTC().Unix(),
		Sha256sum:        outputSha256sum + outExt,
	}

	fileData.Insert()
	r.notifier.NotifyFileAdded(fileData)
	r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
		"type": "convert_image_response",
		"data": map[string]interface{}{
			"file": fileData,
		},
	})
	return nil
}
