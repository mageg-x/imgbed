package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/imgbed/server/utils"
)

// 注册Discord存储驱动
func init() {
	RegisterDriver(StorageTypeDiscord, NewDiscordDriver)
}

// DiscordDriver Discord存储驱动
// 通过Webhook将文件作为附件上传到Discord频道
type DiscordDriver struct {
	webhookURL        string       // Discord Webhook URL
	channelID         string       // 通道ID（用于标识）
	client            *http.Client // HTTP客户端
	channelIDInternal string       // 内部通道ID
}

// DiscordConfig Discord存储配置
type DiscordConfig struct {
	WebhookURL string `json:"webhookUrl"` // Discord Webhook URL
}

// NewDiscordDriver 创建Discord存储驱动实例
// 参数：
//   - cfg: 通道配置
//
// 返回：
//   - StorageDriver: 存储驱动实例
//   - error: 创建失败时的错误
func NewDiscordDriver(cfg *ChannelConfig) (StorageDriver, error) {
	// 从配置中提取Webhook URL
	webhookURL, _ := cfg.Config["webhookUrl"].(string)

	// 验证必需参数
	if webhookURL == "" {
		utils.Errorf("new discord driver: webhook url is required")
		return nil, fmt.Errorf("discord webhook url is required")
	}

	utils.Infof("new discord driver: success")

	return &DiscordDriver{
		webhookURL:        webhookURL,
		channelID:         cfg.ID,
		channelIDInternal: cfg.ID,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Name 返回驱动名称
func (d *DiscordDriver) Name() string {
	return "Discord"
}

// Type 返回存储类型
func (d *DiscordDriver) Type() StorageType {
	return StorageTypeDiscord
}

// Upload 上传文件到Discord
// Discord限制附件最大8MB
// 参数：
//   - ctx: 上下文
//   - req: 上传请求
//
// 返回：
//   - *UploadResult: 上传结果
//   - error: 上传失败时的错误
func (d *DiscordDriver) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	// 读取文件内容
	data, err := io.ReadAll(req.Reader)
	if err != nil {
		utils.Errorf("discord upload: read file failed, error=%v", err)
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	// Discord限制附件最大8MB
	if len(data) > 8*1024*1024 {
		utils.Errorf("discord upload: file size exceeds 8MB limit, size=%d", len(data))
		return nil, fmt.Errorf("file size exceeds 8MB limit for Discord")
	}

	// 生成文件ID
	fileID := req.FileID
	if fileID == "" {
		fileID = generateFileID()
	}

	// 构建multipart表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 创建表单文件字段
	part, err := writer.CreateFormFile("file", req.FileName)
	if err != nil {
		utils.Errorf("discord upload: create form file failed, error=%v", err)
		return nil, fmt.Errorf("create form file failed: %w", err)
	}

	// 写入文件数据
	if _, err := part.Write(data); err != nil {
		utils.Errorf("discord upload: write form file failed, error=%v", err)
		return nil, fmt.Errorf("write form file failed: %w", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		utils.Errorf("discord upload: close writer failed, error=%v", err)
		return nil, fmt.Errorf("close writer failed: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, body)
	if err != nil {
		utils.Errorf("discord upload: create request failed, error=%v", err)
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	resp, err := d.client.Do(httpReq)
	if err != nil {
		utils.Errorf("discord upload: send request failed, error=%v", err)
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		ID          string `json:"id"`
		Attachments []struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			URL      string `json:"url"`
			Size     int    `json:"size"`
		} `json:"attachments"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		utils.Errorf("discord upload: decode response failed, error=%v", err)
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	// 检查是否有附件
	if len(result.Attachments) == 0 {
		utils.Errorf("discord upload: no attachment in response")
		return nil, fmt.Errorf("no attachment in response")
	}

	attachment := result.Attachments[0]

	utils.Debugf("discord upload: success, fileID=%s, attachmentID=%s", fileID, attachment.ID)

	return &UploadResult{
		FileID:    attachment.ID,
		URL:       attachment.URL,
		Size:      int64(attachment.Size),
		ChannelID: d.channelIDInternal,
	}, nil
}

// Download Discord不支持直接下载，返回错误
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - *DownloadResult: 下载结果（始终返回nil）
//   - error: 错误信息
func (d *DiscordDriver) Download(ctx context.Context, fileID string) (*DownloadResult, error) {
	utils.Warnf("discord download: not supported")
	return nil, fmt.Errorf("discord does not support direct download, use URL instead")
}

// GetURL Discord不支持通过文件ID获取URL，返回错误
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - string: 空字符串
//   - error: 错误信息
func (d *DiscordDriver) GetURL(ctx context.Context, fileID string) (string, error) {
	utils.Warnf("discord get url: not supported")
	return "", fmt.Errorf("discord does not support URL retrieval by file ID")
}

// Delete Discord不提供删除API，直接返回成功
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - error: 错误（始终返回nil）
func (d *DiscordDriver) Delete(ctx context.Context, fileID string) error {
	return nil
}

// Exists Discord不提供查询API，直接返回存在
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - bool: 是否存在（始终返回true）
//   - error: 错误（始终返回nil）
func (d *DiscordDriver) Exists(ctx context.Context, fileID string) (bool, error) {
	return true, nil
}

// Stat Discord不支持文件状态查询
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - *FileInfo: 文件信息（始终返回nil）
//   - error: 错误信息
func (d *DiscordDriver) Stat(ctx context.Context, fileID string) (*FileInfo, error) {
	utils.Warnf("discord stat: not supported")
	return nil, fmt.Errorf("discord does not support file stat")
}

// GetQuota 获取存储配额信息（Discord无配额概念）
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - *QuotaInfo: 配额信息
//   - error: 获取失败时的错误
func (d *DiscordDriver) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	return &QuotaInfo{
		UsedSpace:  0,
		TotalSpace: 0,
		FileCount:  0,
	}, nil
}

// HealthCheck 检查Discord Webhook连接状态
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - error: 检查失败时的错误
func (d *DiscordDriver) HealthCheck(ctx context.Context) error {
	// Discord Webhook不支持健康检查，直接返回成功
	return nil
}
