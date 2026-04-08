package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/imgbed/server/utils"
)

// 注册S3和R2驱动
func init() {
	RegisterDriver(StorageTypeS3, NewS3Driver)
	RegisterDriver(StorageTypeR2, NewR2Driver)
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// S3Driver S3兼容存储驱动（支持AWS S3、Cloudflare R2等S3兼容存储）
type S3Driver struct {
	client      *s3.Client  // S3客户端
	bucket      string      // 存储桶名称
	region      string      // 区域
	endpoint    string      // 自定义端点（用于 SDK）
	urlEndpoint string      // URL 访问端点（用于生成访问 URL）
	channelID   string      // 通道ID
	storageType StorageType // 存储类型（S3或R2）
	publicURL   string      // 公共访问URL（R2.dev子域名或自定义域名）
}

// S3Config S3存储配置
type S3Config struct {
	AccessKey string `json:"accessKey"` // Access Key
	SecretKey string `json:"secretKey"` // Secret Key
	Bucket    string `json:"bucket"`    // 存储桶名称
	Region    string `json:"region"`    // 区域
	Endpoint  string `json:"endpoint"`  // 自定义端点（S3兼容服务用）
}

// NewS3Driver 创建S3存储驱动实例
// 参数：
//   - cfg: 通道配置
//
// 返回：
//   - StorageDriver: 存储驱动实例
//   - error: 创建失败时的错误
func NewS3Driver(cfg *ChannelConfig) (StorageDriver, error) {
	// 从配置中提取S3参数
	accessKey, _ := cfg.Config["accessKey"].(string)
	secretKey, _ := cfg.Config["secretKey"].(string)
	bucket, _ := cfg.Config["bucket"].(string)
	region, _ := cfg.Config["region"].(string)
	if region == "" {
		region = "us-east-1"
	}
	endpoint, _ := cfg.Config["endpoint"].(string)

	// 自动处理常见格式问题
	bucket = strings.TrimSpace(bucket)
	endpoint = strings.TrimSpace(endpoint)
	// AWS SDK 要求 endpoint 必须是完整 URI
	if endpoint != "" && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	// 验证必需参数
	if accessKey == "" || secretKey == "" || bucket == "" {
		utils.Errorf("new s3 driver: missing required parameters")
		return nil, fmt.Errorf("s3 access key, secret key and bucket are required")
	}

	// 保存 URL 访问端点（用于生成访问 URL）
	urlEndpoint := endpoint
	// AWS SDK 要求 endpoint 必须是完整 URI
	if endpoint != "" && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	} else if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		// urlEndpoint 需要去掉协议前缀，用于生成 bucket.endpoint 格式的 URL
		urlEndpoint = strings.TrimPrefix(endpoint, "https://")
		urlEndpoint = strings.TrimPrefix(urlEndpoint, "http://")
	}

	// 创建自定义 HTTP 客户端（支持 S3 代理，重新计算签名）
	httpClient := NewS3ProxyHTTPClient(ProxyURLFuncFromConfig(), accessKey, secretKey, region, "s3", 60*time.Second)

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithHTTPClient(httpClient),
	)
	if err != nil {
		utils.Errorf("new s3 driver: load aws config failed, error=%v", err)
		return nil, fmt.Errorf("load aws config failed: %w", err)
	}

	// 创建S3客户端
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	utils.Infof("new s3 driver: success, bucket=%s, region=%s, endpoint=%s", bucket, region, endpoint)

	return &S3Driver{
		client:      client,
		bucket:      bucket,
		region:      region,
		endpoint:    endpoint,
		urlEndpoint: urlEndpoint,
		channelID:   cfg.ID,
		storageType: StorageTypeS3,
	}, nil
}

// NewR2Driver 创建Cloudflare R2存储驱动实例
// 参数：
//   - cfg: 通道配置
//
// 返回：
//   - StorageDriver: 存储驱动实例
//   - error: 创建失败时的错误
func NewR2Driver(cfg *ChannelConfig) (StorageDriver, error) {
	// 从配置中提取R2参数
	accessKey, _ := cfg.Config["accessKey"].(string)
	secretKey, _ := cfg.Config["secretKey"].(string)
	bucket, _ := cfg.Config["bucket"].(string)
	accountID, _ := cfg.Config["accountId"].(string)
	publicURL, _ := cfg.Config["publicUrl"].(string)

	// 调试日志
	utils.Infof("new r2 driver: config keys=%v, accessKey=%s, secretKey=%s, bucket=%s, accountId=%s, publicUrl=%s",
		getKeys(cfg.Config), accessKey != "", secretKey != "", bucket, accountID, publicURL)

	// 验证必需参数
	if accessKey == "" || secretKey == "" || bucket == "" || accountID == "" {
		utils.Errorf("new r2 driver: missing required parameters, accessKey=%v, secretKey=%v, bucket=%v, accountId=%v",
			accessKey != "", secretKey != "", bucket != "", accountID != "")
		return nil, fmt.Errorf("r2 access key, secret key, bucket and account id are required")
	}

	// R2的端点格式
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	// 创建自定义 HTTP 客户端（支持 S3 代理，重新计算签名）
	httpClient := NewS3ProxyHTTPClient(ProxyURLFuncFromConfig(), accessKey, secretKey, "auto", "s3", 60*time.Second)

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithHTTPClient(httpClient),
	)
	if err != nil {
		utils.Errorf("new r2 driver: load aws config failed, error=%v", err)
		return nil, fmt.Errorf("load aws config failed: %w", err)
	}

	// 创建S3客户端
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	utils.Infof("new r2 driver: success, bucket=%s, accountID=%s, publicURL=%s", bucket, accountID, publicURL)

	return &S3Driver{
		client:      client,
		bucket:      bucket,
		region:      "auto",
		endpoint:    endpoint,
		channelID:   cfg.ID,
		storageType: StorageTypeR2,
		publicURL:   publicURL,
	}, nil
}

// Name 返回驱动名称
func (d *S3Driver) Name() string {
	if d.storageType == StorageTypeR2 {
		return "Cloudflare R2"
	}
	return "Amazon S3"
}

// Type 返回存储类型
func (d *S3Driver) Type() StorageType {
	return d.storageType
}

// Upload 上传文件到S3存储
// 参数：
//   - ctx: 上下文
//   - req: 上传请求
//
// 返回：
//   - *UploadResult: 上传结果
//   - error: 上传失败时的错误
func (d *S3Driver) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	// 生成文件ID
	fileID := req.FileID
	if fileID == "" {
		fileID = generateFileID()
	}

	// 添加文件扩展名
	ext := ""
	if idx := strings.LastIndex(req.FileName, "."); idx != -1 {
		ext = strings.ToLower(req.FileName[idx:])
	}
	// fileID 包含扩展名，作为存储的 key
	fileIDWithExt := fileID + ext

	// 构建对象key
	key := fileIDWithExt
	if req.Directory != "" {
		key = req.Directory + "/" + fileIDWithExt
	}

	// 获取 MIME 类型
	mimeType := getMimeType(ext)

	// 上传到S3
	_, err := d.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(d.bucket),
		Key:         aws.String(key),
		Body:        req.Reader,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		utils.Errorf("s3 upload: put object failed, key=%s, error=%v", key, err)
		return nil, fmt.Errorf("put object failed: %w", err)
	}

	// 生成访问URL
	url := d.getObjectURL(key)

	utils.Debugf("s3 upload: success, fileID=%s, key=%s, size=%d", fileIDWithExt, key, req.FileSize)

	return &UploadResult{
		FileID:    fileIDWithExt,
		URL:       url,
		Size:      req.FileSize,
		ChannelID: d.channelID,
	}, nil
}

// Download 从S3存储下载文件
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - *DownloadResult: 下载结果
//   - error: 下载失败时的错误
func (d *S3Driver) Download(ctx context.Context, fileID string) (*DownloadResult, error) {
	key := fileID

	// 从S3获取对象
	result, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		utils.Errorf("s3 download: get object failed, fileID=%s, error=%v", fileID, err)
		return nil, fmt.Errorf("get object failed: %w", err)
	}

	var size int64
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	utils.Debugf("s3 download: success, fileID=%s, size=%d", fileID, size)

	return &DownloadResult{
		Reader:   result.Body,
		Size:     size,
		MimeType: getMimeTypeFromKey(key),
	}, nil
}

// GetURL 获取文件访问URL
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - string: 访问URL
//   - error: 获取失败时的错误
func (d *S3Driver) GetURL(ctx context.Context, fileID string) (string, error) {
	return d.getObjectURL(fileID), nil
}

// Delete 从S3存储删除文件
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - error: 删除失败时的错误
func (d *S3Driver) Delete(ctx context.Context, fileID string) error {
	_, err := d.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(fileID),
	})
	if err != nil {
		utils.Errorf("s3 delete: delete object failed, fileID=%s, error=%v", fileID, err)
	}
	return err
}

// Exists 检查文件是否存在于S3存储
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - bool: 文件是否存在
//   - error: 检查失败时的错误
func (d *S3Driver) Exists(ctx context.Context, fileID string) (bool, error) {
	_, err := d.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(fileID),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Stat 获取S3对象信息
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - *FileInfo: 文件信息
//   - error: 获取失败时的错误
func (d *S3Driver) Stat(ctx context.Context, fileID string) (*FileInfo, error) {
	result, err := d.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(fileID),
	})
	if err != nil {
		utils.Errorf("s3 stat: head object failed, fileID=%s, error=%v", fileID, err)
		return nil, fmt.Errorf("head object failed: %w", err)
	}

	var size int64
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	utils.Debugf("s3 stat: success, fileID=%s, size=%d", fileID, size)

	return &FileInfo{
		FileID:    fileID,
		Size:      size,
		MimeType:  getMimeTypeFromKey(fileID),
		ChannelID: d.channelID,
		CreatedAt: time.Now(),
	}, nil
}

// GetQuota 获取存储配额信息（S3不直接支持配额查询）
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - *QuotaInfo: 配额信息
//   - error: 获取失败时的错误
func (d *S3Driver) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	// S3不提供存储桶使用量查询，需要额外实现
	return &QuotaInfo{
		UsedSpace:  0,
		TotalSpace: 0,
		FileCount:  0,
	}, nil
}

// HealthCheck 检查S3存储健康状态
// 通过 HeadBucket 验证 bucket 是否可访问
func (d *S3Driver) HealthCheck(ctx context.Context) error {
	_, err := d.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(d.bucket),
	})
	if err != nil {
		utils.Errorf("s3 health check: head bucket failed, bucket=%s, error=%v", d.bucket, err)
		return err
	}
	return nil
}

// getObjectURL 生成对象的访问URL
// 参数：
//   - key: 对象key
//
// 返回：
//   - string: 访问URL
func (d *S3Driver) getObjectURL(key string) string {
	if d.storageType == StorageTypeR2 {
		// R2使用配置的公共URL
		if d.publicURL != "" {
			return fmt.Sprintf("%s/%s", strings.TrimSuffix(d.publicURL, "/"), key)
		}
		// 如果没有配置公共URL，返回空字符串，让上层使用本地代理
		return ""
	}

	// S3使用bucket域名格式
	if d.urlEndpoint != "" {
		return fmt.Sprintf("https://%s.%s/%s", d.bucket, d.urlEndpoint, key)
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", d.bucket, d.region, key)
}

// generatePresignedURL 生成预签名URL（暂未使用）
// 参数：
//   - client: S3客户端
//   - bucket: 存储桶名称
//   - key: 对象key
//   - expire: 过期时间
//
// 返回：
//   - string: 预签名URL
//   - error: 生成失败时的错误
func generatePresignedURL(client *s3.Client, bucket, key string, expire time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(client)

	req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expire
	})
	if err != nil {
		return "", err
	}

	return req.URL, nil
}
