package storage

import (
	"net/http"
	"net/url"
	"time"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/utils"
)

type ProxyURLFunc func() string

type ProxyTransport struct {
	ProxyURLFunc ProxyURLFunc
	Base         http.RoundTripper
}

func (t *ProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	proxyURLStr := t.ProxyURLFunc()
	if proxyURLStr == "" {
		base := t.Base
		if base == nil {
			base = http.DefaultTransport
		}
		return base.RoundTrip(req)
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		utils.Errorf("proxy transport: invalid proxy url %s, error=%v", proxyURLStr, err)
		return nil, err
	}

	targetHost := req.URL.Scheme + "://" + req.URL.Host
	encoded := utils.Base58Encode(targetHost)

	proxyReq := req.Clone(req.Context())
	proxyReq.URL.Scheme = proxyURL.Scheme
	proxyReq.URL.Host = proxyURL.Host
	proxyReq.URL.Path = "/proxy/" + encoded + req.URL.Path
	proxyReq.URL.RawQuery = req.URL.RawQuery
	// 设置为代理服务器的 host，否则 Clone 会保留原始请求的 host
	// 导致 Cloudflare 以为我们在访问 api.telegram.org 而返回 403
	proxyReq.Host = proxyURL.Host

	utils.Debugf("proxy transport: %s %s -> %s", req.Method, req.URL.String(), proxyReq.URL.String())

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(proxyReq)
}

func NewProxyHTTPClient(proxyURLFunc ProxyURLFunc, timeout time.Duration) *http.Client {
	transport := &ProxyTransport{
		ProxyURLFunc: proxyURLFunc,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func ProxyURLFuncFromConfig() ProxyURLFunc {
	return func() string {
		cdnConfig := config.GetCDNConfig()
		if cdnConfig.Enabled && cdnConfig.ProxyUrl != "" {
			return cdnConfig.ProxyUrl
		}
		return ""
	}
}

func NoProxyURLFunc() ProxyURLFunc {
	return func() string {
		return ""
	}
}

// S3ProxyTransport S3 专用代理 transport，通过 /s3-proxy 端点重新计算 AWS 签名
type S3ProxyTransport struct {
	ProxyURLFunc ProxyURLFunc
	AccessKey    string
	SecretKey    string
	Region       string
	Service      string
	Base         http.RoundTripper
}

func (t *S3ProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	proxyURLStr := t.ProxyURLFunc()
	if proxyURLStr == "" {
		base := t.Base
		if base == nil {
			base = http.DefaultTransport
		}
		return base.RoundTrip(req)
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		utils.Errorf("s3 proxy transport: invalid proxy url %s, error=%v", proxyURLStr, err)
		return nil, err
	}

	targetURL := req.URL.String()

	proxyReq := req.Clone(req.Context())
	proxyReq.URL.Scheme = proxyURL.Scheme
	proxyReq.URL.Host = proxyURL.Host
	proxyReq.URL.Path = "/s3-proxy"
	proxyReq.URL.RawQuery = ""
	proxyReq.Host = proxyURL.Host
	proxyReq.Header.Set("X-Target-Url", targetURL)
	proxyReq.Header.Set("X-Aws-Access-Key", t.AccessKey)
	proxyReq.Header.Set("X-Aws-Secret-Key", t.SecretKey)
	proxyReq.Header.Set("X-Aws-Region", t.Region)
	proxyReq.Header.Set("X-Aws-Service", t.Service)
	proxyReq.Header.Set("Content-Type", req.Header.Get("Content-Type"))

	utils.Debugf("s3 proxy transport: %s %s -> %s (target: %s)", req.Method, req.URL.String(), proxyReq.URL.String(), targetURL)

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(proxyReq)
}

// NewS3ProxyHTTPClient 创建使用 /s3-proxy 端点的 HTTP 客户端，用于 S3/R2/COS 等需要重新签名的场景
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
