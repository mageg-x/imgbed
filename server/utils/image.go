package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"strings"

	"github.com/deepteams/webp"
	"github.com/disintegration/imaging"
)

const (
	MaxImageMemoryMB   = 256
	MaxImageDimension  = 20000
	MaxImagePixels     = 50000000
	DefaultMaxMemoryMB = 128
)

type CompressionConfig struct {
	Enabled     bool
	Quality     int
	Format      string
	MaxWidth    int
	MaxHeight   int
	MaxMemoryMB int
}

var DefaultCompressionConfig = CompressionConfig{
	Enabled:     true,
	Quality:     80,
	Format:      "webp",
	MaxWidth:    1920,
	MaxHeight:   1080,
	MaxMemoryMB: DefaultMaxMemoryMB,
}

func CompressImage(file *multipart.FileHeader, reader io.Reader, config CompressionConfig) ([]byte, string, error) {
	maxMemory := config.MaxMemoryMB
	if maxMemory <= 0 {
		maxMemory = DefaultMaxMemoryMB
	}
	maxBytes := int64(maxMemory) * 1024 * 1024

	if file.Size > maxBytes {
		Warnf("compress image: file size exceeds memory limit, size=%d, limit=%d, filename=%s",
			file.Size, maxBytes, file.Filename)
		return nil, "", fmt.Errorf("image file too large: %d bytes (max %d MB)", file.Size, maxMemory)
	}

	if !config.Enabled {
		limitedReader := io.LimitReader(reader, maxBytes)
		data, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, "", err
		}
		mimeType := GetMimeType(file)
		return data, mimeType, nil
	}

	limitedReader := io.LimitReader(reader, maxBytes)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", err
	}

	mimeType := GetMimeType(file)
	outputFormat := config.Format
	if outputFormat == "original" {
		outputFormat = strings.TrimPrefix(mimeType, "image/")
	}

	if mimeType == "image/gif" && outputFormat == "gif" {
		return compressGifAnimation(data, config)
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, mimeType, nil
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	pixels := width * height

	if width > MaxImageDimension || height > MaxImageDimension {
		Warnf("compress image: image dimensions too large, width=%d, height=%d, max=%d",
			width, height, MaxImageDimension)
		return nil, "", fmt.Errorf("image dimensions too large: %dx%d (max %d)", width, height, MaxImageDimension)
	}

	if pixels > MaxImagePixels {
		Warnf("compress image: image pixel count too large, pixels=%d, max=%d", pixels, MaxImagePixels)
		return nil, "", fmt.Errorf("image too large: %d pixels (max %d)", pixels, MaxImagePixels)
	}

	if outputFormat == "original" {
		outputFormat = format
	}

	img = resizeImage(img, config.MaxWidth, config.MaxHeight)

	var buf bytes.Buffer
	switch strings.ToLower(outputFormat) {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: config.Quality})
		return buf.Bytes(), "image/jpeg", err
	case "png":
		err = png.Encode(&buf, img)
		return buf.Bytes(), "image/png", err
	case "gif":
		err = gif.Encode(&buf, img, nil)
		return buf.Bytes(), "image/gif", err
	case "webp":
		err = webp.Encode(&buf, img, &webp.EncoderOptions{Quality: float32(config.Quality)})
		return buf.Bytes(), "image/webp", err
	default:
		err = webp.Encode(&buf, img, &webp.EncoderOptions{Quality: float32(config.Quality)})
		return buf.Bytes(), "image/webp", err
	}
}

func compressGifAnimation(data []byte, config CompressionConfig) ([]byte, string, error) {
	originalGif, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return data, "image/gif", nil
	}

	if len(originalGif.Image) == 0 {
		return data, "image/gif", nil
	}

	for i, frame := range originalGif.Image {
		resized := resizeImage(frame, config.MaxWidth, config.MaxHeight)
		if pal, ok := resized.(*image.Paletted); ok {
			originalGif.Image[i] = pal
		}
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, originalGif); err != nil {
		return data, "image/gif", nil
	}

	return buf.Bytes(), "image/gif", nil
}

func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	Infof("resizeImage: input dimensions: width=%d, height=%d, maxWidth=%d, maxHeight=%d", width, height, maxWidth, maxHeight)

	if width <= maxWidth && height <= maxHeight {
		Infof("resizeImage: no resize needed, returning original image")
		return img
	}

	// 计算宽高比，保持等比缩放
	ratio := float64(width) / float64(height)
	var newWidth, newHeight int

	if width > height {
		// 以宽度为基准
		newWidth = maxWidth
		newHeight = int(float64(maxWidth) / ratio)
		if newHeight > maxHeight {
			newHeight = maxHeight
			newWidth = int(float64(maxHeight) * ratio)
		}
	} else {
		// 以高度为基准
		newHeight = maxHeight
		newWidth = int(float64(maxHeight) * ratio)
		if newWidth > maxWidth {
			newWidth = maxWidth
			newHeight = int(float64(maxWidth) / ratio)
		}
	}

	Infof("resizeImage: output dimensions: newWidth=%d, newHeight=%d, originalRatio=%.4f, newRatio=%.4f",
		newWidth, newHeight, ratio, float64(newWidth)/float64(newHeight))

	return imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
}

func IsImageFile(file *multipart.FileHeader) bool {
	mimeType := GetMimeType(file)
	return strings.HasPrefix(mimeType, "image/")
}

func GetImageFormat(file *multipart.FileHeader) string {
	mimeType := GetMimeType(file)
	switch mimeType {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/bmp":
		return "bmp"
	default:
		return "unknown"
	}
}

func CompressImageFromBytes(data []byte, format string, config CompressionConfig) ([]byte, string, error) {
	if !config.Enabled {
		return data, "image/" + format, nil
	}

	if format == "gif" && config.Format == "gif" {
		return compressGifAnimation(data, config)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, "image/" + format, err
	}

	img = resizeImage(img, config.MaxWidth, config.MaxHeight)

	var buf bytes.Buffer
	switch strings.ToLower(config.Format) {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: config.Quality})
		return buf.Bytes(), "image/jpeg", err
	case "png":
		err = png.Encode(&buf, img)
		return buf.Bytes(), "image/png", err
	case "gif":
		err = gif.Encode(&buf, img, nil)
		return buf.Bytes(), "image/gif", err
	case "webp":
		err = webp.Encode(&buf, img, &webp.EncoderOptions{Quality: float32(config.Quality)})
		return buf.Bytes(), "image/webp", err
	default:
		err = webp.Encode(&buf, img, &webp.EncoderOptions{Quality: float32(config.Quality)})
		return buf.Bytes(), "image/webp", err
	}
}

func ValidateImageFormat(format string) bool {
	supported := []string{"jpeg", "jpg", "png", "gif", "webp", "bmp"}
	format = strings.ToLower(format)
	for _, s := range supported {
		if s == format {
			return true
		}
	}
	return false
}

func EstimateWebpSize(img image.Image, quality int) int {
	var buf bytes.Buffer
	webp.Encode(&buf, img, &webp.EncoderOptions{Quality: float32(quality)})
	return buf.Len()
}

func CalculateCompressionRatio(originalSize, compressedSize int64) float64 {
	if originalSize == 0 {
		return 0
	}
	return float64(originalSize-compressedSize) / float64(originalSize) * 100
}
