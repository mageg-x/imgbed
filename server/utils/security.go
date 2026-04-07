package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var magicSignatures = map[string]struct {
	signature []byte
	mimeType  string
}{
	"jpeg":   {[]byte{0xFF, 0xD8, 0xFF}, "image/jpeg"},
	"png":    {[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "image/png"},
	"gif":    {[]byte{0x47, 0x49, 0x46, 0x38}, "image/gif"},
	"webp":   {[]byte{0x52, 0x49, 0x46, 0x46}, "image/webp"},
	"bmp":    {[]byte{0x42, 0x4D}, "image/bmp"},
	"ico":    {[]byte{0x00, 0x00, 0x01, 0x00}, "image/x-icon"},
	"mp4":    {[]byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}, "video/mp4"},
	"webm":   {[]byte{0x1A, 0x45, 0xDF, 0xA3}, "video/webm"},
	"mp3":    {[]byte{0xFF, 0xFB}, "audio/mpeg"},
	"mp3id3": {[]byte{0x49, 0x44, 0x33}, "audio/mpeg"},
	"wav":    {[]byte{0x52, 0x49, 0x46, 0x46}, "audio/wav"},
	"pdf":    {[]byte{0x25, 0x50, 0x44, 0x46}, "application/pdf"},
	"zip":    {[]byte{0x50, 0x4B, 0x03, 0x04}, "application/zip"},
}

func DetectMimeTypeByContent(file *multipart.FileHeader) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()

	header := make([]byte, 512)
	n, err := reader.Read(header)
	if err != nil && err != io.EOF {
		return "", err
	}
	header = header[:n]

	for _, info := range magicSignatures {
		if len(header) >= len(info.signature) {
			if bytes.HasPrefix(header, info.signature) {
				if info.mimeType == "image/webp" {
					if len(header) >= 12 && string(header[8:12]) == "WEBP" {
						return "image/webp", nil
					}
					continue
				}
				if info.mimeType == "audio/wav" {
					if len(header) >= 12 && string(header[8:12]) == "WAVE" {
						return "audio/wav", nil
					}
					continue
				}
				return info.mimeType, nil
			}
		}
	}

	if isSVG(header) {
		return "image/svg+xml", nil
	}

	return "application/octet-stream", nil
}

func isSVG(header []byte) bool {
	content := strings.ToLower(string(header))
	return strings.Contains(content, "<svg") || strings.Contains(content, "<?xml")
}

func ValidateFileType(file *multipart.FileHeader) (string, bool) {
	extMimeType := GetMimeType(file)
	contentMimeType, err := DetectMimeTypeByContent(file)
	if err != nil {
		Errorf("validate file type: detect mime type failed, error=%v", err)
		return extMimeType, false
	}

	if extMimeType == "image/svg+xml" && contentMimeType == "image/svg+xml" {
		return extMimeType, true
	}

	if extMimeType == contentMimeType {
		return extMimeType, true
	}

	if strings.HasPrefix(extMimeType, "image/") && strings.HasPrefix(contentMimeType, "image/") {
		return contentMimeType, true
	}

	if strings.HasPrefix(extMimeType, "video/") && strings.HasPrefix(contentMimeType, "video/") {
		return contentMimeType, true
	}

	if strings.HasPrefix(extMimeType, "audio/") && strings.HasPrefix(contentMimeType, "audio/") {
		return contentMimeType, true
	}

	Warnf("validate file type: mismatch, ext=%s, content=%s, filename=%s", extMimeType, contentMimeType, file.Filename)
	return contentMimeType, false
}

func SanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, "\x00", "")
	filename = strings.ReplaceAll(filename, "..", "")

	var result strings.Builder
	for _, r := range filename {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' || r == ' ' {
			result.WriteRune(r)
		}
	}
	filename = result.String()

	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." {
		filename = "unnamed"
	}

	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		name := filename[:len(filename)-len(ext)]
		if len(name) > 250 {
			name = name[:250]
		}
		filename = name + ext
	}

	return filename
}

func SanitizeDirectory(directory string) string {
	directory = strings.ReplaceAll(directory, "\x00", "")
	directory = strings.ReplaceAll(directory, "..", "")

	directory = strings.TrimPrefix(directory, "/")
	directory = strings.TrimSuffix(directory, "/")

	directory = strings.Trim(directory, " \t\n\r")

	if directory == "." || directory == "" {
		return ""
	}

	var result strings.Builder
	for _, r := range directory {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '/' || r == '-' || r == '_' || r == ' ' {
			result.WriteRune(r)
		}
	}
	directory = result.String()

	parts := strings.Split(directory, "/")
	var cleanParts []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && part != "." {
			cleanParts = append(cleanParts, part)
		}
	}

	return strings.Join(cleanParts, "/")
}

func SanitizeSVG(data []byte) ([]byte, error) {
	content := string(data)

	dangerousPatterns := []string{
		"<script", "</script>", "javascript:", "onerror=", "onload=",
		"onclick=", "onmouseover=", "onfocus=", "onblur=",
		"onkeydown=", "onkeyup=", "onkeypress=",
		"onmousedown=", "onmouseup=", "onmousemove=",
		"onsubmit=", "onreset=", "onchange=", "oninput=",
		"ondblclick=", "oncontextmenu=", "ondrag=", "ondragend=",
		"ondragenter=", "ondragleave=", "ondragover=", "ondragstart=",
		"ondrop=", "onscroll=", "onwheel=", "oncopy=", "oncut=", "onpaste=",
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerContent, strings.ToLower(pattern)) {
			re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(pattern))
			content = re.ReplaceAllString(content, "")
		}
	}

	re := regexp.MustCompile(`(?i)<\s*script[^>]*>.*?<\s*/\s*script\s*>`)
	content = re.ReplaceAllString(content, "")

	re = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["'][^"']*["']`)
	content = re.ReplaceAllString(content, "")

	re = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*[^\s>]+`)
	content = re.ReplaceAllString(content, "")

	re = regexp.MustCompile(`(?i)javascript\s*:`)
	content = re.ReplaceAllString(content, "")

	re = regexp.MustCompile(`(?i)data\s*:\s*text/html`)
	content = re.ReplaceAllString(content, "")

	return []byte(content), nil
}

func IsImageMime(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

func IsVideoMime(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/")
}

func IsAudioMime(mimeType string) bool {
	return strings.HasPrefix(mimeType, "audio/")
}

func IsDocumentMime(mimeType string) bool {
	return mimeType == "application/pdf" || mimeType == "application/zip"
}

func ValidateFileForUpload(file *multipart.FileHeader, allowedTypes []string) (string, error) {
	actualMimeType, typeMatch := ValidateFileType(file)
	if !typeMatch {
		Warnf("validate file for upload: type mismatch, filename=%s, actualType=%s", file.Filename, actualMimeType)
	}

	if !IsAllowedType(actualMimeType, allowedTypes) {
		return actualMimeType, fmt.Errorf("file type not allowed: %s", actualMimeType)
	}

	return actualMimeType, nil
}

// CalcSHA256 计算数据的 SHA256 哈希值
func CalcSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
