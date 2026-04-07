package model

import (
	"strings"
	"time"

	"github.com/imgbed/server/utils"
	"golang.org/x/crypto/bcrypt"
)

// File 文件模型
type File struct {
	ID             string    `gorm:"primaryKey;size:32" json:"id"`
	Name           string    `gorm:"size:255;not null;index" json:"name"`
	OriginalName   string    `gorm:"size:255" json:"originalName"`
	Size           int64     `gorm:"not null" json:"size"`
	Type           string    `gorm:"size:100" json:"type"`
	ChannelID      string    `gorm:"size:32;index:idx_channel_created;index:idx_channel_type" json:"channelId"`
	ChannelType    string    `gorm:"size:20;index" json:"channelType"`
	Directory      string    `gorm:"size:500;index" json:"directory"`
	Tags           string    `gorm:"size:500" json:"tags"`
	AccessCount    int       `gorm:"default:0" json:"accessCount"`
	Checksum       string    `gorm:"size:64;index" json:"checksum"`
	URL            string    `gorm:"size:500" json:"url"`             // 文件访问URL
	UploadedByToken string   `gorm:"size:64;index" json:"uploadedByToken"` // 上传者的Token
	Source         string   `gorm:"size:50;index" json:"source"`         // 来源：user/admin/anonymous/api_xxx
	CreatedAt      time.Time `gorm:"index:idx_channel_created;index" json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (File) TableName() string {
	return "files"
}

// Channel 存储通道模型
type Channel struct {
	ID                string    `gorm:"primaryKey;size:32" json:"id"`
	Name              string    `gorm:"size:100;not null;index" json:"name"`
	Type              string    `gorm:"size:20;not null;index" json:"type"`
	Config            string    `gorm:"type:text" json:"config"`
	Enabled           bool      `gorm:"default:true;index" json:"enabled"`
	Status            string    `gorm:"size:20;default:healthy;index" json:"status"`
	Weight            int       `gorm:"default:100" json:"weight"`
	UsedSpace         int64     `gorm:"default:0" json:"usedSpace"`
	QuotaEnabled      bool      `gorm:"default:false" json:"quotaEnabled"`
	QuotaLimit        int64     `gorm:"default:0" json:"quotaLimit"`
	QuotaThreshold    int       `gorm:"default:90" json:"quotaThreshold"`
	DailyUploadLimit  int       `gorm:"default:0" json:"dailyUploadLimit"`
	DailyUploads      int       `gorm:"default:0" json:"dailyUploads"`
	HourlyUploadLimit int       `gorm:"default:0" json:"hourlyUploadLimit"`
	HourlyUploads     int       `gorm:"default:0" json:"hourlyUploads"`
	MinIntervalMs     int       `gorm:"default:0" json:"minIntervalMs"`
	CooldownMinutes   int       `gorm:"default:60" json:"cooldownMinutes"`
	MaxRetryCount     int       `gorm:"default:3" json:"maxRetryCount"`
	LastUsedAt        time.Time `json:"lastUsedAt"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (Channel) TableName() string {
	return "channels"
}

// Config 系统配置模型
type Config struct {
	Key       string    `gorm:"primaryKey;size:100" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Config) TableName() string {
	return "configs"
}

// APIToken API Token模型
type APIToken struct {
	Token       string    `gorm:"primaryKey;size:64" json:"token"`
	Secret      string    `gorm:"size:64" json:"secret"`
	Name        string    `gorm:"size:100;index" json:"name"`
	Permissions string    `gorm:"size:255" json:"permissions"`
	Enabled     bool      `gorm:"default:true;index" json:"enabled"`
	ExpiresAt   time.Time `gorm:"index" json:"expiresAt"`
	LastUsedAt  time.Time `json:"lastUsedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (APIToken) TableName() string {
	return "api_tokens"
}

// Tag 标签模型
type Tag struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Count     int       `gorm:"default:0" json:"count"`
	CreatedAt time.Time `json:"createdAt"`
}

func (Tag) TableName() string {
	return "tags"
}

// FileAccess 文件访问记录模型
type FileAccess struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FileID     string    `gorm:"size:32;index:idx_file_access;not null" json:"fileId"`
	AccessType string    `gorm:"size:20;index:idx_file_access" json:"accessType"` // upload_success, upload_failed, download, view
	IP         string    `gorm:"size:45;index" json:"ip"`
	UserAgent  string    `gorm:"size:500" json:"userAgent"`
	Referer    string    `gorm:"size:500" json:"referer"`
	AccessAt   time.Time `gorm:"index" json:"accessAt"`
}

func (FileAccess) TableName() string {
	return "file_accesses"
}

// Links 链接格式
type Links struct {
	URL      string `json:"url"`
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
}

// UploadResult 上传结果
type UploadResult struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Size        int64    `json:"size"`
	Type        string   `json:"type"`
	Channel     string   `json:"channel"`
	ChannelType string   `json:"channelType"`
	Directory   string   `json:"directory"`
	Tags        []string `json:"tags"`
	UploadedAt  int64    `json:"uploadedAt"`
	Links       Links    `json:"links"`
}

// ChannelConfig 通道配置
type ChannelConfig struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Enabled   bool                   `json:"enabled"`
	Config    map[string]interface{} `json:"config"`
	Quota     QuotaConfig            `json:"quota"`
	RateLimit RateLimitConfig        `json:"rateLimit"`
}

// QuotaConfig 配额配置
type QuotaConfig struct {
	Enabled   bool `json:"enabled"`
	LimitGB   int  `json:"limitGB"`
	Threshold int  `json:"threshold"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	DailyUploadLimit  int `json:"dailyUploadLimit"`
	HourlyUploadLimit int `json:"hourlyUploadLimit"`
	MinIntervalMs     int `json:"minIntervalMs"`
}

// FileInfo 文件信息
type FileInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	URL          string   `json:"url"`
	Size         int64    `json:"size"`
	Type         string   `json:"type"`
	Channel      string   `json:"channel"`
	ChannelType  string   `json:"channelType"`
	Directory    string   `json:"directory"`
	Tags         []string `json:"tags"`
	UploadedAt   int64    `json:"uploadedAt"`
	AccessCount  int      `json:"accessCount"`
	LastAccessAt int64    `json:"lastAccessAt"`
	Checksum     string   `json:"checksum"`
	Links        Links    `json:"links"`
}

// Directory 目录信息
type Directory struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	FileCount int    `json:"fileCount"`
}

// ChannelStatus 通道状态
type ChannelStatus struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Enabled      bool   `json:"enabled"`
	Status       string `json:"status"`
	UsedSpace    int64  `json:"usedSpace"`
	TotalSpace   int64  `json:"totalSpace"`
	UsagePercent int    `json:"usagePercent"`
	FileCount    int    `json:"fileCount"`
}

// ChannelQuota 通道配额信息
type ChannelQuota struct {
	ChannelID     string `json:"channelId"`
	UsedSpace     int64  `json:"usedSpace"`
	TotalSpace    int64  `json:"totalSpace"`
	UsagePercent  int    `json:"usagePercent"`
	FileCount     int    `json:"fileCount"`
	DailyUploads  int    `json:"dailyUploads"`
	DailyLimit    int    `json:"dailyLimit"`
	HourlyUploads int    `json:"hourlyUploads"`
	HourlyLimit   int    `json:"hourlyLimit"`
}

// StatsOverview 统计概览
type StatsOverview struct {
	TotalFiles   int            `json:"totalFiles"`
	TotalSize    int64          `json:"totalSize"`
	TodayUploads int            `json:"todayUploads"`
	TodayTraffic int64          `json:"todayTraffic"`
	ChannelStats []ChannelStats `json:"channelStats"`
}

// ChannelStats 通道统计
type ChannelStats struct {
	ChannelID    string `json:"channelId"`
	FileCount    int    `json:"fileCount"`
	UsedSpace    int64  `json:"usedSpace"`
	UsagePercent int    `json:"usagePercent"`
}

// HashPassword 使用bcrypt加密密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		utils.Errorf("hash password: generate failed, error=%v", err)
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash 验证密码
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		utils.Debugf("check password hash: mismatch")
	}
	return err == nil
}

// IsImageMime 判断MIME类型是否为图片
func IsImageMime(mimeType string) bool {
	return strings.HasPrefix(strings.ToLower(mimeType), "image/")
}
