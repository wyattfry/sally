package web

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// saveUploadedFile reads a multipart file from the request, saves it to
// uploadsDir with a hex-encoded random filename, and returns the URL path
// (e.g. /uploads/<guid>.<ext>). Returns ("", nil) when no file was uploaded.
func saveUploadedFile(r *http.Request, fieldName, uploadsDir string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err == http.ErrMissingFile {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read uploaded file: %w", err)
	}
	defer file.Close()

	// Sniff the first 512 bytes to determine the real MIME type.
	sniff := make([]byte, 512)
	n, err := file.Read(sniff)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read file header: %w", err)
	}
	sniff = sniff[:n]
	ext := mimeToExt(http.DetectContentType(sniff))
	if ext == "" {
		if e := filepath.Ext(header.Filename); e != "" {
			ext = e
		} else {
			ext = ".bin"
		}
	}

	guidBytes := make([]byte, 16)
	if _, err := rand.Read(guidBytes); err != nil {
		return "", fmt.Errorf("generate guid: %w", err)
	}
	filename := hex.EncodeToString(guidBytes) + ext

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("create uploads dir: %w", err)
	}

	dst, err := os.Create(filepath.Join(uploadsDir, filename))
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := dst.Write(sniff); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return "/uploads/" + filename, nil
}

var imageDownloadClient = &http.Client{Timeout: 20 * time.Second}

// downloadAndSave fetches an external image URL, scales it down to max 800px,
// saves it locally as JPEG, and returns the local URL path. Returns the original
// URL unchanged if anything fails.
func downloadAndSave(imageURL, uploadsDir string) string {
	if !strings.HasPrefix(imageURL, "http") || uploadsDir == "" {
		return imageURL
	}

	resp, err := imageDownloadClient.Get(imageURL)
	if err != nil || resp.StatusCode >= 400 {
		if err == nil {
			resp.Body.Close()
		}
		log.Printf("image download %s: skipped (err=%v status=%v)", imageURL, err, func() int {
			if err == nil {
				return resp.StatusCode
			}
			return 0
		}())
		return imageURL
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return imageURL
	}

	if http.DetectContentType(data) == "application/octet-stream" {
		return imageURL
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return imageURL
	}

	scaled := scaleDown(img, 800)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: 85}); err != nil {
		return imageURL
	}

	guidBytes := make([]byte, 16)
	if _, err := rand.Read(guidBytes); err != nil {
		return imageURL
	}
	filename := hex.EncodeToString(guidBytes) + ".jpg"

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return imageURL
	}
	if err := os.WriteFile(filepath.Join(uploadsDir, filename), buf.Bytes(), 0644); err != nil {
		return imageURL
	}

	return "/uploads/" + filename
}

func scaleDown(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return src
	}
	var newW, newH int
	if w > h {
		newW = maxDim
		newH = max(1, h*maxDim/w)
	} else {
		newH = maxDim
		newW = max(1, w*maxDim/h)
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := range newH {
		for x := range newW {
			dst.Set(x, y, src.At(b.Min.X+x*w/newW, b.Min.Y+y*h/newH))
		}
	}
	return dst
}

func mimeToExt(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tiff"
	default:
		return ""
	}
}
