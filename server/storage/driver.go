package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"time"
)

// 存储类型常量定义
type StorageType string

const (
	StorageTypeLocal       StorageType = "local"       // 本地存储
	StorageTypeTelegram    StorageType = "telegram"    // Telegram存储
	StorageTypeR2          StorageType = "cfr2"        // Cloudflare R2存储
	StorageTypeS3          StorageType = "s3"          // AWS S3存储
	StorageTypeDiscord     StorageType = "discord"     // Discord存储
	StorageTypeHuggingFace StorageType = "huggingface" // HuggingFace存储
)

// UploadRequest 上传请求结构
type UploadRequest struct {
	FileID    string    // 文件唯一ID
	FileName  string    // 文件名
	FileSize  int64     // 文件大小（字节）
	Reader    io.Reader // 文件内容读取器
	Directory string    // 存储目录
	Tags      []string  // 文件标签
	ChannelID string    // 通道ID
}

// UploadResult 上传结果结构
type UploadResult struct {
	FileID    string     // 文件ID
	URL       string     // 访问URL
	Size      int64      // 文件大小
	ChannelID string     // 通道ID
	ExpiresAt *time.Time // URL过期时间（可选）
}

// DownloadResult 下载结果结构
type DownloadResult struct {
	Reader   io.ReadCloser // 文件内容读取器
	Size     int64         // 文件大小
	MimeType string        // MIME类型
}

// FileInfo 文件信息结构
type FileInfo struct {
	FileID    string    // 文件ID
	Name      string    // 文件名
	Size      int64     // 文件大小
	MimeType  string    // MIME类型
	ChannelID string    // 通道ID
	CreatedAt time.Time // 创建时间
}

// QuotaInfo 配额信息结构
type QuotaInfo struct {
	UsedSpace  int64 // 已用空间（字节）
	TotalSpace int64 // 总空间（字节，0表示无限制）
	FileCount  int   // 文件数量
}

// StorageDriver 存储驱动接口
// 所有存储驱动必须实现此接口
type StorageDriver interface {
	// Name 返回驱动名称
	Name() string
	// Type 返回存储类型
	Type() StorageType
	// Upload 上传文件
	Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error)
	// Download 下载文件
	Download(ctx context.Context, fileID string) (*DownloadResult, error)
	// GetURL 获取文件访问URL
	GetURL(ctx context.Context, fileID string) (string, error)
	// Delete 删除文件
	Delete(ctx context.Context, fileID string) error
	// Exists 检查文件是否存在
	Exists(ctx context.Context, fileID string) (bool, error)
	// Stat 获取文件信息
	Stat(ctx context.Context, fileID string) (*FileInfo, error)
	// GetQuota 获取存储配额信息
	GetQuota(ctx context.Context) (*QuotaInfo, error)
	// HealthCheck 检查存储服务健康状态
	HealthCheck(ctx context.Context) error
}

// ChannelConfig 通道配置结构
type ChannelConfig struct {
	ID              string                 // 通道ID
	Name            string                 // 通道名称
	Type            StorageType            // 存储类型
	Enabled         bool                   // 是否启用
	Config          map[string]interface{} // 驱动配置
	QuotaEnabled    bool                   // 是否启用配额限制
	QuotaLimit      int64                  // 配额限制（字节）
	QuotaThreshold  int                    // 配额阈值（百分比）
	DailyLimit      int                    // 日上传限制
	HourlyLimit     int                    // 小时上传限制
	MinIntervalMs   int                    // 最小上传间隔（毫秒）
	CooldownMinutes int                    // 冷却时间（分钟）
	MaxRetryCount   int                    // 最大重试次数
	Weight          int                    // 负载均衡权重
}

// DriverFactory 驱动工厂函数类型
type DriverFactory func(config *ChannelConfig) (StorageDriver, error)

// driverFactories 存储驱动工厂注册表
var driverFactories = make(map[StorageType]DriverFactory)

// RegisterDriver 注册存储驱动工厂
// 在init()函数中调用，将驱动注册到全局注册表
func RegisterDriver(storageType StorageType, factory DriverFactory) {
	driverFactories[storageType] = factory
}

// CreateDriver 根据配置创建存储驱动实例
// 参数：
//   - config: 通道配置
//
// 返回：
//   - StorageDriver: 存储驱动实例
//   - error: 创建失败时的错误
func CreateDriver(config *ChannelConfig) (StorageDriver, error) {
	factory, ok := driverFactories[config.Type]
	if !ok {
		return nil, ErrUnsupportedStorageType
	}
	return factory(config)
}

// 存储相关错误定义
var (
	// ErrUnsupportedStorageType 不支持的存储类型
	ErrUnsupportedStorageType = &StorageError{Code: 30001, Message: "unsupported storage type"}
	// ErrChannelUnavailable 通道不可用
	ErrChannelUnavailable = &StorageError{Code: 30001, Message: "channel unavailable"}
	// ErrChannelQuotaFull 通道配额已满
	ErrChannelQuotaFull = &StorageError{Code: 30002, Message: "channel quota full"}
	// ErrUploadFailed 上传失败
	ErrUploadFailed = &StorageError{Code: 30003, Message: "upload failed"}
	// ErrFileNotFound 文件不存在
	ErrFileNotFound = &StorageError{Code: 10001, Message: "file not found"}
)

// StorageError 存储操作错误类型
type StorageError struct {
	Code    int    // 错误码
	Message string // 错误信息
}

// Error 实现error接口
func (e *StorageError) Error() string {
	return e.Message
}

// generateFileID 生成唯一的文件ID
// 返回：16字节随机字符串的十六进制表示
func generateFileID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getMimeTypeFromKey 从文件key/路径中推断MIME类型
// 参数：
//   - key: 文件key或路径
//
// 返回：
//   - string: 推断的MIME类型
func getMimeTypeFromKey(key string) string {
	// 从后往前查找最后一个点号
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			return getMimeType(key[i:])
		}
	}
	return "application/octet-stream"
}

// getMimeType 根据文件扩展名获取MIME类型
// 参数：
//   - ext: 文件扩展名（如".jpg", ".png"）
//
// 返回：
//   - string: MIME类型
func getMimeType(ext string) string {
	// 常见文件类型的MIME类型映射
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
	}

	// 查找预定义的MIME类型
	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	// 尝试使用http.DetectContentType检测
	mimeType := http.DetectContentType([]byte(ext))
	if mimeType != "application/octet-stream" {
		return mimeType
	}

	// 默认返回octet-stream
	return "application/octet-stream"
}
