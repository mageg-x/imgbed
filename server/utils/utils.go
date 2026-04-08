package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateID 生成唯一的32字符ID
// 使用加密安全的随机数生成器，返回16字节的十六进制字符串
// 返回：32字符的唯一ID字符串
func GenerateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateShortID 生成短ID（8字符）
// 用于需要较短标识符的场景
// 返回：8字符的短ID字符串
func GenerateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetFileExtension 获取文件扩展名（小写）
// 参数：
//   - filename: 文件名
//
// 返回：
//   - string: 小写的文件扩展名（包含点号，如".jpg"）
func GetFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// GetMimeType 根据文件扩展名获取MIME类型
// 参数：
//   - file: 上传的文件头信息
//
// 返回：
//   - string: MIME类型字符串
func GetMimeType(file *multipart.FileHeader) string {
	ext := GetFileExtension(file.Filename)
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".bmp":  "image/bmp",
		".ico":  "image/x-icon",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".zip": "application/zip",
		".rar": "application/x-rar-compressed",
		".7z":  "application/x-7z-compressed",
		".txt":  "text/plain",
		".json": "application/json",
		".xml":  "application/xml",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
	}

	f, err := file.Open()
	if err == nil {
		defer f.Close()
		buffer := make([]byte, 512)
		n, err := f.Read(buffer)
		if err == nil && n > 0 {
			detected := http.DetectContentType(buffer[:n])
			if detected != "application/octet-stream" {
				return detected
			}
		}
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return file.Header.Get("Content-Type")
}

// IsAllowedType 检查MIME类型是否在允许列表中
// 参数：
//   - mimeType: 待检查的MIME类型
//   - allowedTypes: 允许的MIME类型列表（支持通配符，如"image/*"）
//
// 返回：
//   - bool: 是否允许
func IsAllowedType(mimeType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		// 默认只允许图片类型
		return strings.HasPrefix(mimeType, "image/")
	}
	for _, allowed := range allowedTypes {
		// 支持通配符匹配，如"image/*"匹配所有图片类型
		if strings.HasSuffix(allowed, "/*") {
			prefix := strings.TrimSuffix(allowed, "/*")
			if strings.HasPrefix(mimeType, prefix) {
				return true
			}
		} else if mimeType == allowed {
			return true
		}
	}
	return false
}

// FormatSize 格式化文件大小为人类可读格式
// 参数：
//   - bytes: 文件大小（字节）
//
// 返回：
//   - string: 格式化后的大小字符串（如"1.50 MB"）
func FormatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	sizes := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	fb := float64(bytes)
	for fb >= unit && i < len(sizes)-1 {
		fb /= unit
		i++
	}
	return fmt.Sprintf("%.2f %s", fb, sizes[i])
}

// SaveUploadedFile 保存上传的文件到指定路径
// 参数：
//   - file: 上传的文件头信息
//   - dst: 目标保存路径
//
// 返回：
//   - error: 保存失败时的错误
func SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	// 打开源文件
	src, err := file.Open()
	if err != nil {
		logLogger.Errorf("save uploaded file: open failed, error=%v", err)
		return err
	}
	defer src.Close()

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		logLogger.Errorf("save uploaded file: create directory failed, error=%v", err)
		return err
	}

	// 创建目标文件
	out, err := os.Create(dst)
	if err != nil {
		logLogger.Errorf("save uploaded file: create file failed, error=%v", err)
		return err
	}
	defer out.Close()

	// 复制文件内容
	_, err = io.Copy(out, src)
	if err != nil {
		logLogger.Errorf("save uploaded file: copy failed, error=%v", err)
	}
	return err
}

// FileExists 检查文件是否存在
// 参数：
//   - path: 文件路径
//
// 返回：
//   - bool: 文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// DeleteFile 删除指定文件
// 参数：
//   - path: 文件路径
//
// 返回：
//   - error: 删除失败时的错误
func DeleteFile(path string) error {
	return os.Remove(path)
}

// GetClientIP 从HTTP请求中获取客户端真实IP
// 依次检查X-Forwarded-For、X-Real-IP头，最后使用RemoteAddr
// 参数：
//   - r: HTTP请求
//
// 返回：
//   - string: 客户端IP地址
func GetClientIP(r *http.Request) string {
	// 优先检查X-Forwarded-For头（代理场景）
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	// 检查X-Real-IP头（Nginx代理场景）
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	// 最后使用RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

// ParseTags 解析标签字符串为标签数组
// 参数：
//   - tagsStr: 逗号分隔的标签字符串
//
// 返回：
//   - []string: 标签数组（已去除空白）
func ParseTags(tagsStr string) []string {
	if tagsStr == "" {
		return []string{}
	}
	tags := strings.Split(tagsStr, ",")
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

// TagsToString 将标签数组转换为逗号分隔的字符串
// 参数：
//   - tags: 标签数组
//
// 返回：
//   - string: 逗号分隔的标签字符串
func TagsToString(tags []string) string {
	return strings.Join(tags, ",")
}

// GetCurrentTimestamp 获取当前Unix时间戳（秒）
// 返回：
//   - int64: 当前时间戳
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// GetCurrentTimestampMs 获取当前Unix时间戳（毫秒）
// 返回：
//   - int64: 当前时间戳（毫秒）
func GetCurrentTimestampMs() int64 {
	return time.Now().UnixMilli()
}

// NormalizeDirectory 规范化目录路径
// 去除前后斜杠，确保目录以斜杠结尾
// 参数：
//   - dir: 原始目录路径
//
// 返回：
//   - string: 规范化后的目录路径
func NormalizeDirectory(dir string) string {
	dir = strings.TrimSpace(dir)
	dir = strings.TrimPrefix(dir, "/")
	dir = strings.TrimSuffix(dir, "/")
	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}

// GetFileHash 计算文件哈希值（预留接口）
// 参数：
//   - file: 文件读取器
//
// 返回：
//   - string: 文件哈希值
//   - error: 计算失败时的错误
func GetFileHash(file io.Reader) (string, error) {
	return "", nil
}

// EnsureDir 确保目录存在，不存在则创建
// 参数：
//   - dir: 目录路径
//
// 返回：
//   - error: 创建失败时的错误
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
