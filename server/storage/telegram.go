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

// 注册Telegram存储驱动
func init() {
	RegisterDriver(StorageTypeTelegram, NewTelegramDriver)
}

// TelegramDriver Telegram存储驱动
// 将文件作为文档上传到Telegram频道
type TelegramDriver struct {
	botToken          string       // Bot Token
	channelID         string       // 主频道ID
	channelID2        string       // 备用频道ID（用于负载均衡）
	client            *http.Client // HTTP客户端
	channelIDInternal string       // 内部通道ID
}

// TelegramConfig Telegram存储配置
type TelegramConfig struct {
	BotToken   string `json:"botToken"`   // Bot Token
	ChannelID  string `json:"channelId"`  // 主频道ID
	ChannelID2 string `json:"channelId2"` // 备用频道ID
}

// NewTelegramDriver 创建Telegram存储驱动实例
// 参数：
//   - cfg: 通道配置
//
// 返回：
//   - StorageDriver: 存储驱动实例
//   - error: 创建失败时的错误
func NewTelegramDriver(cfg *ChannelConfig) (StorageDriver, error) {
	// 从配置中提取Telegram参数
	botToken, _ := cfg.Config["botToken"].(string)
	channelID, _ := cfg.Config["channelId"].(string)
	channelID2, _ := cfg.Config["channelId2"].(string)

	// 验证必需参数
	if botToken == "" || channelID == "" {
		utils.Errorf("new telegram driver: missing required parameters")
		return nil, fmt.Errorf("telegram bot token and channel id are required")
	}

	utils.Infof("new telegram driver: success, channelID=%s", channelID)

	return &TelegramDriver{
		botToken:          botToken,
		channelID:         channelID,
		channelID2:        channelID2,
		channelIDInternal: cfg.ID,
		client:            NewProxyHTTPClient(ProxyURLFuncFromConfig(), 60*time.Second),
	}, nil
}

// Name 返回驱动名称
func (d *TelegramDriver) Name() string {
	return "Telegram"
}

// Type 返回存储类型
func (d *TelegramDriver) Type() StorageType {
	return StorageTypeTelegram
}

// getApiBaseUrl 获取 Telegram API 基础 URL
func (d *TelegramDriver) getApiBaseUrl() string {
	return fmt.Sprintf("https://api.telegram.org/bot%s", d.botToken)
}

// getFileBaseUrl 获取 Telegram 文件下载基础 URL
func (d *TelegramDriver) getFileBaseUrl() string {
	return fmt.Sprintf("https://api.telegram.org/file/bot%s", d.botToken)
}

// Upload 上传文件到Telegram
// 文件作为文档上传，支持最大20MB
// 参数：
//   - ctx: 上下文
//   - req: 上传请求
//
// 返回：
//   - *UploadResult: 上传结果
//   - error: 上传失败时的错误
func (d *TelegramDriver) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	// 读取文件内容
	data, err := io.ReadAll(req.Reader)
	if err != nil {
		utils.Errorf("telegram upload: read file failed, error=%v", err)
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	// Telegram限制文件最大20MB
	if len(data) > 20*1024*1024 {
		utils.Errorf("telegram upload: file size exceeds 20MB limit, size=%d", len(data))
		return nil, fmt.Errorf("file size exceeds 20MB limit for Telegram")
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
	part, err := writer.CreateFormFile("document", req.FileName)
	if err != nil {
		utils.Errorf("telegram upload: create form file failed, error=%v", err)
		return nil, fmt.Errorf("create form file failed: %w", err)
	}

	// 写入文件数据
	if _, err := part.Write(data); err != nil {
		utils.Errorf("telegram upload: write form file failed, error=%v", err)
		return nil, fmt.Errorf("write form file failed: %w", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		utils.Errorf("telegram upload: close writer failed, error=%v", err)
		return nil, fmt.Errorf("close writer failed: %w", err)
	}

	// 选择频道（奇偶负载均衡）
	channelID := d.channelID
	if d.channelID2 != "" && time.Now().Unix()%2 == 0 {
		channelID = d.channelID2
	}

	// 构建请求URL
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/sendDocument?chat_id=%s", apiBase, channelID)

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		utils.Errorf("telegram upload: create request failed, error=%v", err)
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	resp, err := d.client.Do(httpReq)
	if err != nil {
		utils.Errorf("telegram upload: send request failed, error=%v", err)
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
			Document  struct {
				FileID   string `json:"file_id"`
				FileName string `json:"file_name"`
				FileSize int64  `json:"file_size"`
			} `json:"document"`
		} `json:"result"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		utils.Errorf("telegram upload: decode response failed, error=%v", err)
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	if !result.OK {
		utils.Errorf("telegram upload: telegram api error, description=%s", result.Description)
		return nil, fmt.Errorf("telegram api error: %s", result.Description)
	}

	tgFileID := result.Result.Document.FileID

	// 获取文件路径（需要额外调用 getFile API）
	filePath, err := d.getFilePath(ctx, tgFileID)
	if err != nil {
		utils.Warnf("telegram upload: get file path failed, fileID=%s, error=%v", tgFileID, err)
		// 即使获取失败也继续，但 URL 可能不正确
		filePath = tgFileID
	}

	utils.Debugf("telegram upload: success, fileID=%s, tgFileID=%s, filePath=%s", fileID, tgFileID, filePath)

	return &UploadResult{
		FileID:    tgFileID,
		URL:       fmt.Sprintf("%s/%s", d.getFileBaseUrl(), filePath),
		Size:      result.Result.Document.FileSize,
		ChannelID: d.channelIDInternal,
	}, nil
}

// Download 从Telegram下载文件
// 参数：
//   - ctx: 上下文
//   - fileID: Telegram文件ID
//
// 返回：
//   - *DownloadResult: 下载结果
//   - error: 下载失败时的错误
func (d *TelegramDriver) Download(ctx context.Context, fileID string) (*DownloadResult, error) {
	// 获取文件路径
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/getFile?file_id=%s", apiBase, fileID)

	resp, err := d.client.Get(url)
	if err != nil {
		utils.Errorf("telegram download: get file info failed, error=%v", err)
		return nil, fmt.Errorf("get file info failed: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
			FileSize int64  `json:"file_size"`
		} `json:"result"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		utils.Errorf("telegram download: decode response failed, error=%v", err)
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	if !result.OK {
		utils.Errorf("telegram download: telegram api error, description=%s", result.Description)
		return nil, fmt.Errorf("telegram api error: %s", result.Description)
	}

	// 下载文件内容
	fileURL := fmt.Sprintf("%s/%s", d.getFileBaseUrl(), result.Result.FilePath)
	fileResp, err := d.client.Get(fileURL)
	if err != nil {
		utils.Errorf("telegram download: download file failed, error=%v", err)
		return nil, fmt.Errorf("download file failed: %w", err)
	}

	utils.Debugf("telegram download: success, fileID=%s", fileID)

	return &DownloadResult{
		Reader:   fileResp.Body,
		Size:     result.Result.FileSize,
		MimeType: getMimeTypeFromPath(result.Result.FilePath),
	}, nil
}

// GetURL 获取Telegram文件的访问URL
// 参数：
//   - ctx: 上下文
//   - fileID: Telegram文件ID
//
// 返回：
//   - string: 访问URL
//   - error: 获取失败时的错误
func (d *TelegramDriver) GetURL(ctx context.Context, fileID string) (string, error) {
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/getFile?file_id=%s", apiBase, fileID)

	resp, err := d.client.Get(url)
	if err != nil {
		utils.Errorf("telegram get url: get file info failed, error=%v", err)
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		utils.Errorf("telegram get url: decode response failed, error=%v", err)
		return "", err
	}

	if !result.OK {
		utils.Errorf("telegram get url: file not found, fileID=%s", fileID)
		return "", ErrFileNotFound
	}

	return fmt.Sprintf("%s/%s", d.getFileBaseUrl(), result.Result.FilePath), nil
}

// Delete 删除Telegram文件（Telegram不提供删除API，直接返回成功）
// 参数：
//   - ctx: 上下文
//   - fileID: Telegram文件ID
//
// 返回：
//   - error: 错误（始终返回nil）
func (d *TelegramDriver) Delete(ctx context.Context, fileID string) error {
	// Telegram Bot API不提供删除文件的接口
	return nil
}

// Exists 检查文件是否存在于Telegram
// 参数：
//   - ctx: 上下文
//   - fileID: Telegram文件ID
//
// 返回：
//   - bool: 文件是否存在
//   - error: 检查失败时的错误
func (d *TelegramDriver) Exists(ctx context.Context, fileID string) (bool, error) {
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/getFile?file_id=%s", apiBase, fileID)

	resp, err := d.client.Get(url)
	if err != nil {
		utils.Errorf("telegram exists: get file info failed, error=%v", err)
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		OK bool `json:"ok"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		utils.Errorf("telegram exists: decode response failed, error=%v", err)
		return false, err
	}

	return result.OK, nil
}

// Stat 获取Telegram文件信息
// 参数：
//   - ctx: 上下文
//   - fileID: Telegram文件ID
//
// 返回：
//   - *FileInfo: 文件信息
//   - error: 获取失败时的错误
func (d *TelegramDriver) Stat(ctx context.Context, fileID string) (*FileInfo, error) {
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/getFile?file_id=%s", apiBase, fileID)

	resp, err := d.client.Get(url)
	if err != nil {
		utils.Errorf("telegram stat: get file info failed, error=%v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
			FileSize int64  `json:"file_size"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		utils.Errorf("telegram stat: decode response failed, error=%v", err)
		return nil, err
	}

	if !result.OK {
		utils.Errorf("telegram stat: file not found, fileID=%s", fileID)
		return nil, ErrFileNotFound
	}

	utils.Debugf("telegram stat: success, fileID=%s", fileID)

	return &FileInfo{
		FileID:    fileID,
		Size:      result.Result.FileSize,
		MimeType:  getMimeTypeFromPath(result.Result.FilePath),
		ChannelID: d.channelIDInternal,
	}, nil
}

// GetQuota 获取存储配额信息（Telegram无配额概念）
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - *QuotaInfo: 配额信息
//   - error: 获取失败时的错误
func (d *TelegramDriver) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	return &QuotaInfo{
		UsedSpace:  0,
		TotalSpace: 0,
		FileCount:  0,
	}, nil
}

// HealthCheck 检查Telegram Bot连接状态
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - error: 检查失败时的错误
func (d *TelegramDriver) HealthCheck(ctx context.Context) error {
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/getMe", apiBase)

	resp, err := d.client.Get(url)
	if err != nil {
		utils.Errorf("telegram health check: get me failed, error=%v", err)
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK bool `json:"ok"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		utils.Errorf("telegram health check: decode response failed, error=%v", err)
		return err
	}

	if !result.OK {
		utils.Errorf("telegram health check: bot token invalid")
		return fmt.Errorf("telegram bot token invalid")
	}

	return nil
}

// getMimeTypeFromPath 从文件路径推断MIME类型
// 参数：
//   - path: 文件路径
//
// 返回：
//   - string: MIME类型
func getMimeTypeFromPath(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return getMimeType(path[i:])
		}
	}
	return "application/octet-stream"
}

// getFilePath 调用 getFile API 获取文件的路径
// 参数：
//   - ctx: 上下文
//   - fileID: Telegram 文件 ID
//
// 返回：
//   - string: 文件路径
//   - error: 获取失败时的错误
func (d *TelegramDriver) getFilePath(ctx context.Context, fileID string) (string, error) {
	apiBase := d.getApiBaseUrl()
	url := fmt.Sprintf("%s/getFile?file_id=%s", apiBase, fileID)

	resp, err := d.client.Get(url)
	if err != nil {
		utils.Errorf("telegram getFilePath: request failed, error=%v", err)
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		utils.Errorf("telegram getFilePath: decode failed, error=%v", err)
		return "", err
	}

	if !result.OK {
		utils.Errorf("telegram getFilePath: api error, description=%s", result.Description)
		return "", fmt.Errorf("telegram api error: %s", result.Description)
	}

	return result.Result.FilePath, nil
}
