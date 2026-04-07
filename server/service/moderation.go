package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/imgbed/server/model"
	"github.com/imgbed/server/utils"
)

// ModerationService 内容审核服务，负责图片内容的安全审核
type ModerationService struct {
	configService *ConfigService // 配置服务引用
}

// NewModerationService 创建ModerationService实例
func NewModerationService() *ModerationService {
	return &ModerationService{
		configService: NewConfigService(),
	}
}

// ModerationResult 审核结果结构
type ModerationResult struct {
	Safe            bool     `json:"safe"`                      // 是否安全
	Labels          []string `json:"labels,omitempty"`          // 检测到的标签
	Confidence      float64  `json:"confidence,omitempty"`      // 置信度
	Provider        string   `json:"provider"`                  // 审核服务提供商
	ErrorMessage    string   `json:"errorMessage,omitempty"`    // 错误信息
	SuggestedAction string   `json:"suggestedAction,omitempty"` // 建议操作
}

// ModerationProvider 内容审核提供者接口
type ModerationProvider interface {
	Check(ctx context.Context, data []byte, filename string) (*ModerationResult, error)
	Name() string
}

// CheckContent 对图片内容进行安全审核
// 参数：
//   - ctx: 上下文
//   - data: 图片数据
//   - filename: 文件名
//
// 返回：
//   - *ModerationResult: 审核结果
//   - error: 审核过程中的错误
func (s *ModerationService) CheckContent(ctx context.Context, data []byte, filename string) (*ModerationResult, error) {
	// 获取审核配置
	cfg, err := s.configService.GetModerationConfig()
	if err != nil {
		utils.Errorf("check content: get moderation config failed, error=%v", err)
		return nil, fmt.Errorf("failed to get moderation config: %w", err)
	}

	// 如果审核未启用，直接返回安全
	if !cfg.Enabled {
		return &ModerationResult{
			Safe:     true,
			Provider: "disabled",
		}, nil
	}

	// 获取审核提供者
	provider, err := s.getProvider(cfg.Provider, cfg.APIKey)
	if err != nil {
		utils.Errorf("check content: get provider failed, provider=%s, error=%v", cfg.Provider, err)
		return nil, err
	}

	return provider.Check(ctx, data, filename)
}

// getProvider 根据名称获取审核提供者实例
// 参数：
//   - name: 提供者名称（aws、google、aliyun、tencent）
//   - apiKey: API密钥
//
// 返回：
//   - ModerationProvider: 审核提供者实例
//   - error: 获取过程中的错误
func (s *ModerationService) getProvider(name, apiKey string) (ModerationProvider, error) {
	switch name {
	case "aws":
		return NewAWSModerationProvider(apiKey), nil
	case "google":
		return NewGoogleModerationProvider(apiKey), nil
	case "aliyun":
		return NewAliyunModerationProvider(apiKey), nil
	case "tencent":
		return NewTencentModerationProvider(apiKey), nil
	default:
		utils.Errorf("get provider: unsupported provider, name=%s", name)
		return nil, fmt.Errorf("unsupported moderation provider: %s", name)
	}
}

// ShouldModerate 判断MIME类型是否需要审核
// 参数：
//   - mimeType: 文件的MIME类型
//
// 返回：
//   - bool: 是否需要审核
func (s *ModerationService) ShouldModerate(mimeType string) bool {
	return model.IsImageMime(mimeType)
}

// AWSModerationProvider AWS审核提供者实现
type AWSModerationProvider struct {
	apiKey string
	client *http.Client
}

// NewAWSModerationProvider 创建AWS审核提供者实例
func NewAWSModerationProvider(apiKey string) *AWSModerationProvider {
	return &AWSModerationProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name 返回提供者名称
func (p *AWSModerationProvider) Name() string {
	return "aws"
}

// Check 执行AWS图片审核（当前为占位实现）
func (p *AWSModerationProvider) Check(ctx context.Context, data []byte, filename string) (*ModerationResult, error) {
	// AWS Rekognition API集成占位
	// 当前实现直接返回安全状态
	return &ModerationResult{
		Safe:            true,
		Provider:        p.Name(),
		Confidence:      0.99,
		SuggestedAction: "allow",
	}, nil
}

// GoogleModerationProvider Google审核提供者实现
type GoogleModerationProvider struct {
	apiKey string
	client *http.Client
}

// NewGoogleModerationProvider 创建Google审核提供者实例
func NewGoogleModerationProvider(apiKey string) *GoogleModerationProvider {
	return &GoogleModerationProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name 返回提供者名称
func (p *GoogleModerationProvider) Name() string {
	return "google"
}

// Check 执行Google Vision API图片审核
func (p *GoogleModerationProvider) Check(ctx context.Context, data []byte, filename string) (*ModerationResult, error) {
	// 如果未配置API Key，直接返回安全
	if p.apiKey == "" {
		return &ModerationResult{
			Safe:            true,
			Provider:        p.Name(),
			Confidence:      1.0,
			SuggestedAction: "allow",
		}, nil
	}

	// 构建Google Vision API请求URL
	url := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", p.apiKey)

	// 构建请求体
	requestBody := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"image": map[string]interface{}{
					"content": data,
				},
				"features": []map[string]interface{}{
					{"type": "SAFE_SEARCH_DETECTION"},
				},
			},
		},
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		utils.Errorf("google moderation check: marshal request failed, error=%v", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		utils.Errorf("google moderation check: create request failed, error=%v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		utils.Errorf("google moderation check: send request failed, error=%v", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Errorf("google moderation check: read response failed, error=%v", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		utils.Errorf("google moderation check: API error, status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	// 解析响应结果
	var result struct {
		Responses []struct {
			SafeSearchAnnotation struct {
				Adult    string `json:"adult"`
				Spoof    string `json:"spoof"`
				Medical  string `json:"medical"`
				Violence string `json:"violence"`
				Racy     string `json:"racy"`
			} `json:"safeSearchAnnotation"`
		} `json:"responses"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		utils.Errorf("google moderation check: parse response failed, error=%v", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 如果没有响应结果，返回安全
	if len(result.Responses) == 0 {
		return &ModerationResult{
			Safe:            true,
			Provider:        p.Name(),
			Confidence:      1.0,
			SuggestedAction: "allow",
		}, nil
	}

	annotation := result.Responses[0].SafeSearchAnnotation
	labels := []string{}
	isUnsafe := false

	// 检测成人内容
	if annotation.Adult == "LIKELY" || annotation.Adult == "VERY_LIKELY" {
		labels = append(labels, "adult")
		isUnsafe = true
	}
	// 检测暴力内容
	if annotation.Violence == "LIKELY" || annotation.Violence == "VERY_LIKELY" {
		labels = append(labels, "violence")
		isUnsafe = true
	}
	// 检测暴露内容
	if annotation.Racy == "LIKELY" || annotation.Racy == "VERY_LIKELY" {
		labels = append(labels, "racy")
	}
	// 检测医疗内容
	if annotation.Medical == "LIKELY" || annotation.Medical == "VERY_LIKELY" {
		labels = append(labels, "medical")
	}
	// 检测恶搞内容
	if annotation.Spoof == "LIKELY" || annotation.Spoof == "VERY_LIKELY" {
		labels = append(labels, "spoof")
	}

	// 确定建议操作
	action := "allow"
	if isUnsafe {
		action = "reject"
	} else if len(labels) > 0 {
		action = "review"
	}

	return &ModerationResult{
		Safe:            !isUnsafe,
		Labels:          labels,
		Confidence:      0.9,
		Provider:        p.Name(),
		SuggestedAction: action,
	}, nil
}

// AliyunModerationProvider 阿里云审核提供者实现
type AliyunModerationProvider struct {
	apiKey string
	client *http.Client
}

// NewAliyunModerationProvider 创建阿里云审核提供者实例
func NewAliyunModerationProvider(apiKey string) *AliyunModerationProvider {
	return &AliyunModerationProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name 返回提供者名称
func (p *AliyunModerationProvider) Name() string {
	return "aliyun"
}

// Check 执行阿里云图片审核（当前为占位实现）
func (p *AliyunModerationProvider) Check(ctx context.Context, data []byte, filename string) (*ModerationResult, error) {
	// 阿里云内容安全API集成占位
	// 当前实现直接返回安全状态
	return &ModerationResult{
		Safe:            true,
		Provider:        p.Name(),
		Confidence:      0.99,
		SuggestedAction: "allow",
	}, nil
}

// TencentModerationProvider 腾讯云审核提供者实现
type TencentModerationProvider struct {
	apiKey string
	client *http.Client
}

// NewTencentModerationProvider 创建腾讯云审核提供者实例
func NewTencentModerationProvider(apiKey string) *TencentModerationProvider {
	return &TencentModerationProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name 返回提供者名称
func (p *TencentModerationProvider) Name() string {
	return "tencent"
}

// Check 执行腾讯云图片审核（当前为占位实现）
func (p *TencentModerationProvider) Check(ctx context.Context, data []byte, filename string) (*ModerationResult, error) {
	// 腾讯云内容安全API集成占位
	// 当前实现直接返回安全状态
	return &ModerationResult{
		Safe:            true,
		Provider:        p.Name(),
		Confidence:      0.99,
		SuggestedAction: "allow",
	}, nil
}
