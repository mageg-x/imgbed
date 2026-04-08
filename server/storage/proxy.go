package storage

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/utils"
)

// ==================== 类型定义 ====================

// ProxyURLFunc 返回代理服务器 URL 的函数类型
// 返回空字符串表示不使用代理，直接请求目标服务器
type ProxyURLFunc func() string

// ProxyTransport HTTP 代理传输器
//
// 功能：
// - 将请求转发到 Cloudflare Worker 代理
// - 通过 Base58 编码混淆目标主机名
// - 支持 Telegram、Discord 等存储后端的上传请求
//
// URL 格式: {proxy_url}/proxy/{base58_encoded_host}{original_path}
type ProxyTransport struct {
	ProxyURLFunc ProxyURLFunc // 获取代理 URL 的函数
	Base         http.RoundTripper // 基础传输器，为 nil 时使用 http.DefaultTransport
}

// S3ProxyTransport S3/R2/COS 专用代理传输器
//
// 功能：
// - 将 S3 签名请求转发到 Cloudflare Worker
// - Worker 使用 aws4fetch 重新计算 AWS SigV4 签名
// - 支持 R2 (Cloudflare)、COS (腾讯云)、S3 (AWS) 等对象存储
//
// URL 格式: {proxy_url}/s3-proxy
// 请求头：
//   - X-Target-Url: 目标 S3 URL
//   - X-Aws-Access-Key: AWS Access Key
//   - X-Aws-Secret-Key: AWS Secret Key
//   - X-Aws-Region: AWS 区域
//   - X-Aws-Service: 服务名（默认 s3）
type S3ProxyTransport struct {
	ProxyURLFunc ProxyURLFunc
	AccessKey    string // AWS Access Key
	SecretKey    string // AWS Secret Key
	Region       string // AWS 区域
	Service      string // AWS 服务名
	Base         http.RoundTripper
}

// ==================== ProxyTransport ====================

// RoundTrip 实现 http.RoundTripper 接口
//
// 流程：
// 1. 获取代理 URL，为空则直接发送原始请求
// 2. 解析代理 URL
// 3. 对目标主机进行 Base58 编码
// 4. 构造代理请求 URL: /proxy/{encoded_host}{original_path}
// 5. 设置 proxyReq.Host 为代理服务器主机（避免 Cloudflare 403）
// 6. 发送请求
func (t *ProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 获取代理 URL
	proxyURLStr := t.ProxyURLFunc()

	// 不使用代理，直接发送原始请求
	if proxyURLStr == "" {
		base := t.Base
		if base == nil {
			base = http.DefaultTransport
		}
		return base.RoundTrip(req)
	}

	// 解析代理 URL
	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		utils.Errorf("proxy transport: invalid proxy url %s, error=%v", proxyURLStr, err)
		return nil, err
	}

	// 对目标主机进行 Base58 编码
	// 例如: https://api.telegram.org -> AXJnNj1p7gNFB24iwS1NQ1rucwwcJEtSS
	targetHost := req.URL.Scheme + "://" + req.URL.Host
	encoded := utils.Base58Encode(targetHost)

	// 克隆请求并修改 URL
	proxyReq := req.Clone(req.Context())
	proxyReq.URL.Scheme = proxyURL.Scheme
	proxyReq.URL.Host = proxyURL.Host
	proxyReq.URL.Path = "/proxy/" + encoded + req.URL.Path
	proxyReq.URL.RawQuery = req.URL.RawQuery

	// 设置 Host 为代理服务器主机，避免 Cloudflare 403
	// 如果不设置，Clone 会保留原始请求的 Host (api.telegram.org)
	// 导致 Cloudflare 认为我们在访问 api.telegram.org 而返回 403
	proxyReq.Host = proxyURL.Host

	utils.Debugf("proxy transport: %s %s -> %s", req.Method, req.URL.String(), proxyReq.URL.String())

	// 发送请求
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(proxyReq)
}

// ==================== S3ProxyTransport ====================

// RoundTrip 实现 http.RoundTripper 接口
//
// 流程：
// 1. 获取代理 URL，为空则直接发送原始请求
// 2. 解析代理 URL
// 3. 提取原始请求 URL 作为目标 URL
// 4. 构造代理请求，路径为 /s3-proxy
// 5. 在请求头中传递 AWS 凭证和目标 URL
// 6. Worker 重新计算 SigV4 签名后转发到 S3
func (t *S3ProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 获取代理 URL
	proxyURLStr := t.ProxyURLFunc()

	// 不使用代理，直接发送原始请求
	if proxyURLStr == "" {
		base := t.Base
		if base == nil {
			base = http.DefaultTransport
		}
		return base.RoundTrip(req)
	}

	// 解析代理 URL
	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		utils.Errorf("s3 proxy transport: invalid proxy url %s, error=%v", proxyURLStr, err)
		return nil, err
	}

	// 目标 S3 URL
	targetURL := req.URL.String()

	// 克隆请求并修改 URL
	proxyReq := req.Clone(req.Context())
	proxyReq.URL.Scheme = proxyURL.Scheme
	proxyReq.URL.Host = proxyURL.Host
	proxyReq.URL.Path = "/s3-proxy"
	proxyReq.URL.RawQuery = ""
	proxyReq.Host = proxyURL.Host

	// 设置 AWS 凭证到请求头，供 Worker 重新签名
	proxyReq.Header.Set("X-Target-Url", targetURL)
	proxyReq.Header.Set("X-Aws-Access-Key", t.AccessKey)
	proxyReq.Header.Set("X-Aws-Secret-Key", t.SecretKey)
	proxyReq.Header.Set("X-Aws-Region", t.Region)
	proxyReq.Header.Set("X-Aws-Service", t.Service)
	proxyReq.Header.Set("Content-Type", req.Header.Get("Content-Type"))

	utils.Debugf("s3 proxy transport: %s %s -> %s (target: %s)", req.Method, req.URL.String(), proxyReq.URL.String(), targetURL)

	// 发送请求
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(proxyReq)
}

// ==================== 构造函数 ====================

// NewProxyHTTPClient 创建使用 ProxyTransport 的 HTTP 客户端
// 用于 Telegram、Discord 等存储后端的上传请求
func NewProxyHTTPClient(proxyURLFunc ProxyURLFunc, timeout time.Duration) *http.Client {
	transport := &ProxyTransport{
		ProxyURLFunc: proxyURLFunc,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// NewS3ProxyHTTPClient 创建使用 S3ProxyTransport 的 HTTP 客户端
// 用于 S3、R2、COS 等需要重新签名的对象存储
func NewS3ProxyHTTPClient(proxyURLFunc ProxyURLFunc, accessKey, secretKey, region, service string, timeout time.Duration) *http.Client {
	transport := &S3ProxyTransport{
		ProxyURLFunc: proxyURLFunc,
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		Region:       region,
		Service:      service,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// ==================== 辅助函数 ====================

// ProxyURLFuncFromConfig 从配置获取代理 URL
// 当 CDN 配置启用且设置了 ProxyUrl 时返回该 URL，否则返回空字符串
func ProxyURLFuncFromConfig() ProxyURLFunc {
	return func() string {
		cdnConfig := config.GetCDNConfig()
		if cdnConfig.Enabled && cdnConfig.ProxyUrl != "" {
			return cdnConfig.ProxyUrl
		}
		return ""
	}
}

// NoProxyURLFunc 返回始终为空字符串的 ProxyURLFunc
// 用于禁用代理，直接连接目标服务器
func NoProxyURLFunc() ProxyURLFunc {
	return func() string {
		return ""
	}
}

// ReadResponseBody 读取响应体，自动处理 gzip 解压
// 用于处理代理服务器可能返回的 gzip 压缩响应
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 尝试直接解析 JSON，失败则尝试 gzip 解压后再解析
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Encoding")), "gzip") {
		gzReader, err := gzip.NewReader(bytes.NewReader(bodyBytes))
		if err == nil {
			defer gzReader.Close()
			return io.ReadAll(gzReader)
		}
	}

	return bodyBytes, nil
}
