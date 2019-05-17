package admin

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/edwvee/exiffix"
	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
	"github.com/nfnt/resize"
)

// TODO: configurable max image width.
var (
	MaxImageWidth = 1280
	JpegQuality   = 90
)

// processImage manhandles an image's binary data, scaling it down to <= 1280
// pixels and stripping off any metadata.
func processImage(input []byte, ext string) ([]byte, error) {
	if ext == ".gif" {
		return input, nil
	}

	reader := bytes.NewReader(input)

	// Decode the image using exiffix, which will auto-rotate jpeg images etc.
	// based on their EXIF values.
	origImage, _, err := exiffix.Decode(reader)
	if err != nil {
		return input, err
	}

	// Read the config to get the image width.
	reader.Seek(0, io.SeekStart)
	config, _, _ := image.DecodeConfig(reader)
	width := config.Width

	// If the width is too great, scale it down.
	if width > MaxImageWidth {
		width = MaxImageWidth
	}
	newImage := resize.Resize(uint(width), 0, origImage, resize.Lanczos3)

	var output bytes.Buffer
	switch ext {
	case ".jpeg":
		fallthrough
	case ".jpg":
		jpeg.Encode(&output, newImage, &jpeg.Options{
			Quality: JpegQuality,
		})
	case ".png":
		png.Encode(&output, newImage)
	case ".gif":
		gif.Encode(&output, newImage, nil)
	}

	return output.Bytes(), nil
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer

	type response struct {
		Success  bool   `json:"success"`
		Error    string `json:"error,omitempty"`
		Filename string `json:"filename,omitempty"`
		URI      string `json:"uri,omitempty"`
		Checksum string `json:"checksum,omitempty"`
	}

	// Get the file from the form data.
	file, header, err := r.FormFile("file")
	if err != nil {
		responses.JSON(w, http.StatusBadRequest, response{
			Error: err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate the extension is an image type.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		responses.JSON(w, http.StatusBadRequest, response{
			Error: "Invalid file type, only common image types are supported: jpg, png, gif",
		})
		return
	}

	// Read the file.
	io.Copy(&buf, file)
	binary := buf.Bytes()

	// Process and image and resize it down, strip metadata, etc.
	binary, err = processImage(binary, ext)
	if err != nil {
		responses.JSON(w, http.StatusBadRequest, response{
			Error: "Resize error: " + err.Error(),
		})
	}

	// Make a checksum of it.
	sha := sha256.New()
	sha.Write(binary)
	checksum := hex.EncodeToString(sha.Sum(nil))

	log.Info("Uploaded file names: %s   Checksum is: %s", header.Filename, checksum)

	// Write to the /static/photos directory of the user root. Ensure the path
	// exists or create it if not.
	outputPath := filepath.Join(*render.UserRoot, "static", "photos")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		os.MkdirAll(outputPath, 0755)
	}

	// Write the output file.
	filename := filepath.Join(outputPath, checksum+ext)
	outfh, err := os.Create(filename)
	if err != nil {
		responses.JSON(w, http.StatusBadRequest, response{
			Error: err.Error(),
		})
		return
	}
	defer outfh.Close()
	outfh.Write(binary)

	responses.JSON(w, http.StatusOK, response{
		Success:  true,
		Filename: header.Filename,
		URI:      fmt.Sprintf("/static/photos/%s%s", checksum, ext),
		Checksum: checksum,
	})
}
