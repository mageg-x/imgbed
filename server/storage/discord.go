package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/utils"
)

// 注册Discord存储驱动
func init() {
	RegisterDriver(StorageTypeDiscord, NewDiscordDriver)
}

// DiscordDriver Discord存储驱动
// 通过 Bot Token 将文件作为附件上传到 Discord 频道
type DiscordDriver struct {
	botToken          string       // Discord Bot Token
	channelID         string       // Discord 频道 ID
	isNitro           bool         // 是否 Nitro 会员（25MB vs 8MB）
	client            *http.Client // HTTP客户端
	channelIDInternal string       // 内部通道ID
}

// NewDiscordDriver 创建Discord存储驱动实例
// 参数：
//   - cfg: 通道配置
//
// 返回：
//   - StorageDriver: 存储驱动实例
//   - error: 创建失败时的错误
func NewDiscordDriver(cfg *ChannelConfig) (StorageDriver, error) {
	botToken, _ := cfg.Config["botToken"].(string)
	channelID, _ := cfg.Config["channelId"].(string)
	isNitro, _ := cfg.Config["isNitro"].(bool)

	if botToken == "" || channelID == "" {
		utils.Errorf("new discord driver: bot token and channel id are required")
		return nil, fmt.Errorf("discord bot token and channel id is required")
	}

	utils.Infof("new discord driver: success, channelID=%s, isNitro=%v", channelID, isNitro)

	return &DiscordDriver{
		botToken:          botToken,
		channelID:         channelID,
		isNitro:           isNitro,
		channelIDInternal: cfg.ID,
		client:            NewProxyHTTPClient(ProxyURLFuncFromConfig(), 60*time.Second),
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

// getApiBaseUrl 获取 Discord API 基础 URL
func (d *DiscordDriver) getApiBaseUrl() string {
	return "https://discord.com/api/v10"
}

// getFileSizeLimit 获取文件大小限制
func (d *DiscordDriver) getFileSizeLimit() int {
	if d.isNitro {
		return 25 * 1024 * 1024 // 25MB for Nitro
	}
	return 8 * 1024 * 1024 // 8MB for free tier
}

// Upload 上传文件到 Discord
// Discord 限制附件最大 8MB（Nitro 25MB）
func (d *DiscordDriver) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	data, err := io.ReadAll(req.Reader)
	if err != nil {
		utils.Errorf("discord upload: read file failed, error=%v", err)
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	sizeLimit := d.getFileSizeLimit()
	if len(data) > sizeLimit {
		utils.Errorf("discord upload: file size exceeds %dMB limit, size=%d", sizeLimit/1024/1024, len(data))
		return nil, fmt.Errorf("file size exceeds %dMB limit for Discord", sizeLimit/1024/1024)
	}

	fileID := req.FileID
	if fileID == "" {
		fileID = generateFileID()
	}

	// 构建 multipart 表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 创建表单文件字段（Discord 使用 files[0]）
	part, err := writer.CreateFormFile("files[0]", req.FileName)
	if err != nil {
		utils.Errorf("discord upload: create form file failed, error=%v", err)
		return nil, fmt.Errorf("create form file failed: %w", err)
	}

	if _, err := part.Write(data); err != nil {
		utils.Errorf("discord upload: write form file failed, error=%v", err)
		return nil, fmt.Errorf("write form file failed: %w", err)
	}

	if err := writer.Close(); err != nil {
		utils.Errorf("discord upload: close writer failed, error=%v", err)
		return nil, fmt.Errorf("close writer failed: %w", err)
	}

	// 构建请求
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/channels/%s/messages", apiBase, d.channelID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		utils.Errorf("discord upload: create request failed, error=%v", err)
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.botToken))
	httpReq.Header.Set("User-Agent", "DiscordBot (ImgBed, 1.0)")

	resp, err := d.client.Do(httpReq)
	if err != nil {
		utils.Errorf("discord upload: send request failed, error=%v", err)
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体（自动处理 gzip 解压）
	bodyBytes, err := ReadResponseBody(resp)
	if err != nil {
		utils.Errorf("discord upload: read body failed, error=%v", err)
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var result struct {
		ID          string `json:"id"`
		Attachments []struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			URL      string `json:"url"`
			ProxyURL string `json:"proxy_url"`
			Size     int    `json:"size"`
		} `json:"attachments"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		utils.Errorf("discord upload: decode response failed, body=%s, error=%v", string(bodyBytes)[:min(200, len(bodyBytes))], err)
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	if len(result.Attachments) == 0 {
		utils.Errorf("discord upload: no attachment in response")
		return nil, fmt.Errorf("no attachment in response")
	}

	attachment := result.Attachments[0]

	utils.Debugf("discord upload: success, fileID=%s, messageID=%s, attachmentID=%s", fileID, result.ID, attachment.ID)

	fileID = fmt.Sprintf("%s:%s:%s", d.channelID, result.ID, attachment.ID)

	var fileUrl string
	proxyUrl := config.GetCDNProxyUrl()
	if proxyUrl != "" {
		fileUrl = d.generateProxyURL(result.ID, attachment.ID, req.FileName)
	} else {
		fileUrl = attachment.URL
	}

	return &UploadResult{
		FileID:    fileID,
		URL:       fileUrl,
		Size:      int64(attachment.Size),
		ChannelID: d.channelIDInternal,
	}, nil
}

// Download Discord 不支持直接下载
func (d *DiscordDriver) Download(ctx context.Context, fileID string) (*DownloadResult, error) {
	utils.Warnf("discord download: not supported")
	return nil, fmt.Errorf("discord does not support direct download, use URL instead")
}

// GetURL Discord 不支持通过文件ID获取URL
func (d *DiscordDriver) GetURL(ctx context.Context, fileID string) (string, error) {
	utils.Warnf("discord get url: not supported")
	return "", fmt.Errorf("discord does not support URL retrieval by file ID")
}

// Delete 删除 Discord 消息（删除文件）
func (d *DiscordDriver) Delete(ctx context.Context, fileID string) error {
	parts := strings.Split(fileID, ":")
	if len(parts) < 2 {
		utils.Warnf("discord delete: invalid fileID format, fileID=%s", fileID)
		return fmt.Errorf("invalid fileID format")
	}

	channelID := parts[0]
	messageID := parts[1]

	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/channels/%s/messages/%s", apiBase, channelID, messageID)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		utils.Errorf("discord delete: create request failed, error=%v", err)
		return fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.botToken))
	httpReq.Header.Set("User-Agent", "DiscordBot (ImgBed, 1.0)")

	resp, err := d.client.Do(httpReq)
	if err != nil {
		utils.Errorf("discord delete: send request failed, fileID=%s, error=%v", fileID, err)
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	// 204 No Content 表示成功
	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		utils.Debugf("discord delete: success, fileID=%s", fileID)
		return nil
	}

	utils.Errorf("discord delete: failed, fileID=%s, status=%d", fileID, resp.StatusCode)
	return fmt.Errorf("delete failed: status %d", resp.StatusCode)
}

// Exists Discord 不提供查询 API
func (d *DiscordDriver) Exists(ctx context.Context, fileID string) (bool, error) {
	return true, nil
}

// Stat Discord 不支持文件状态查询
func (d *DiscordDriver) Stat(ctx context.Context, fileID string) (*FileInfo, error) {
	utils.Warnf("discord stat: not supported")
	return nil, fmt.Errorf("discord does not support file stat")
}

// GetQuota 获取存储配额信息
func (d *DiscordDriver) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	return &QuotaInfo{
		UsedSpace:  0,
		TotalSpace: 0,
		FileCount:  0,
	}, nil
}

// HealthCheck 检查 Discord Bot 连接状态
func (d *DiscordDriver) HealthCheck(ctx context.Context) error {
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/users/@me", apiBase)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.botToken))
	httpReq.Header.Set("User-Agent", "DiscordBot (ImgBed, 1.0)")

	resp, err := d.client.Do(httpReq)
	if err != nil {
		utils.Errorf("discord health check: request failed, error=%v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		utils.Errorf("discord health check: bot token invalid, status=%d", resp.StatusCode)
		return fmt.Errorf("bot token invalid")
	}

	return nil
}

func (d *DiscordDriver) generateProxyURL(messageID, attachmentID, fileName string) string {
	proxyUrl := strings.TrimSuffix(config.GetCDNProxyUrl(), "/")
	payload := d.botToken + "|" + d.channelID + "|" + messageID + "|" + attachmentID
	encrypted, err := utils.EncryptTelegramPayload(payload)
	if err != nil {
		utils.Warnf("discord generateProxyURL: encrypt failed, error=%v", err)
		return ""
	}
	encoded := utils.Base58EncodeBytes(encrypted)
	discordPayload := "discord:" + encoded
	finalEncoded := utils.Base58Encode(discordPayload)
	ext := filepath.Ext(fileName)
	return fmt.Sprintf("%s/%s/%s", proxyUrl, finalEncoded, ext)
}
