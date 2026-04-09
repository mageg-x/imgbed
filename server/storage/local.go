package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/imgbed/server/utils"
)

// 注册本地存储驱动
func init() {
	RegisterDriver(StorageTypeLocal, NewLocalDriver)
}

// LocalDriver 本地存储驱动
// 将文件存储在本地文件系统中
type LocalDriver struct {
	basePath  string // 存储根目录
	channelID string // 通道ID
}

// LocalConfig 本地存储配置
type LocalConfig struct {
	Path string `json:"path"` // 自定义存储路径
}

// NewLocalDriver 创建本地存储驱动实例
// 参数：
//   - cfg: 通道配置
//
// 返回：
//   - StorageDriver: 存储驱动实例
//   - error: 创建失败时的错误
func NewLocalDriver(cfg *ChannelConfig) (StorageDriver, error) {
	// 默认路径：平台特定位置
	var basePath string
	if path, ok := cfg.Config["path"].(string); ok && path != "" {
		basePath = path
	} else {
		// 平台特定默认路径（与数据库目录一致）
		switch runtime.GOOS {
		case "windows":
			configDir, _ := os.UserConfigDir()
			basePath = filepath.Join(configDir, "ImgBed", "uploads")
		case "darwin":
			configDir, _ := os.UserConfigDir()
			basePath = filepath.Join(configDir, "ImgBed", "uploads")
		default:
			// Linux: ~/.imgbed/uploads
			home, _ := os.UserHomeDir()
			basePath = filepath.Join(home, ".imgbed", "uploads")
		}
	}

	// 确保存储目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		utils.Errorf("new local driver: create directory failed, path=%s, error=%v", basePath, err)
		return nil, fmt.Errorf("create local storage directory failed: %w", err)
	}

	return &LocalDriver{
		basePath:  basePath,
		channelID: cfg.ID,
	}, nil
}

// Name 返回驱动名称
func (d *LocalDriver) Name() string {
	return "Local Storage"
}

// Type 返回存储类型
func (d *LocalDriver) Type() StorageType {
	return StorageTypeLocal
}

// Upload 上传文件到本地存储
// 参数：
//   - ctx: 上下文
//   - req: 上传请求
//
// 返回：
//   - *UploadResult: 上传结果
//   - error: 上传失败时的错误
func (d *LocalDriver) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	// 生成文件ID
	fileID := req.FileID
	if fileID == "" {
		fileID = generateFileID()
	}

	// 获取文件扩展名
	ext := filepath.Ext(req.FileName)
	// 构建相对路径
	relativePath := fileID + ext
	if req.Directory != "" {
		relativePath = filepath.Join(req.Directory, fileID+ext)
	}

	// 构建完整路径
	fullPath := filepath.Join(d.basePath, relativePath)
	dir := filepath.Dir(fullPath)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		utils.Errorf("local upload: create directory failed, path=%s, error=%v", dir, err)
		return nil, fmt.Errorf("create directory failed: %w", err)
	}

	// 创建目标文件
	file, err := os.Create(fullPath)
	if err != nil {
		utils.Errorf("local upload: create file failed, path=%s, error=%v", fullPath, err)
		return nil, fmt.Errorf("create file failed: %w", err)
	}
	defer file.Close()

	// 复制文件内容
	written, err := io.Copy(file, req.Reader)
	if err != nil {
		// 复制失败，删除已创建的文件
		os.Remove(fullPath)
		utils.Errorf("local upload: write file failed, path=%s, error=%v", fullPath, err)
		return nil, fmt.Errorf("write file failed: %w", err)
	}

	// 构建访问URL（相对路径，由服务器代理）
	fileURL := "/api/v1/file/" + fileID + ext

	utils.Debugf("local upload: success, fileID=%s, path=%s, url=%s, size=%d", fileID+ext, fullPath, fileURL, written)

	return &UploadResult{
		FileID:    fileID + ext,
		URL:       fileURL,
		Size:      written,
		ChannelID: d.channelID,
	}, nil
}

// Download 从本地存储下载文件
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - *DownloadResult: 下载结果
//   - error: 下载失败时的错误
func (d *LocalDriver) Download(ctx context.Context, fileID string) (*DownloadResult, error) {
	// 查找文件（支持任意扩展名）
	paths, err := filepath.Glob(filepath.Join(d.basePath, "**", fileID+"*"))
	if err != nil {
		utils.Errorf("local download: glob failed, fileID=%s, error=%v", fileID, err)
		return nil, err
	}

	// 如果没找到，尝试直接在根目录查找
	if len(paths) == 0 {
		paths, err = filepath.Glob(filepath.Join(d.basePath, fileID+"*"))
		if err != nil {
			utils.Errorf("local download: glob failed, fileID=%s, error=%v", fileID, err)
			return nil, err
		}
	}

	// 文件不存在
	if len(paths) == 0 {
		utils.Warnf("local download: file not found, fileID=%s", fileID)
		return nil, ErrFileNotFound
	}

	filePath := paths[0]
	file, err := os.Open(filePath)
	if err != nil {
		utils.Errorf("local download: open file failed, path=%s, error=%v", filePath, err)
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		utils.Errorf("local download: stat file failed, path=%s, error=%v", filePath, err)
		return nil, err
	}

	utils.Debugf("local download: success, fileID=%s, path=%s, size=%d", fileID, filePath, stat.Size())

	return &DownloadResult{
		Reader:   file,
		Size:     stat.Size(),
		MimeType: getMimeType(filepath.Ext(filePath)),
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
func (d *LocalDriver) GetURL(ctx context.Context, fileID string) (string, error) {
	return "/api/v1/file/" + fileID, nil
}

// Delete 删除本地文件
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - error: 删除失败时的错误
func (d *LocalDriver) Delete(ctx context.Context, fileID string) error {
	// 查找文件
	paths, err := filepath.Glob(filepath.Join(d.basePath, "**", fileID+"*"))
	if err != nil {
		utils.Errorf("local delete: glob failed, fileID=%s, error=%v", fileID, err)
		return err
	}

	// 如果没找到，尝试直接在根目录查找
	if len(paths) == 0 {
		paths, err = filepath.Glob(filepath.Join(d.basePath, fileID+"*"))
		if err != nil {
			utils.Errorf("local delete: glob failed, fileID=%s, error=%v", fileID, err)
			return err
		}
	}

	// 删除所有匹配的文件
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			utils.Errorf("local delete: remove file failed, path=%s, error=%v", p, err)
			return err
		}
	}

	utils.Debugf("local delete: success, fileID=%s", fileID)
	return nil
}

// Exists 检查文件是否存在
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - bool: 文件是否存在
//   - error: 检查失败时的错误
func (d *LocalDriver) Exists(ctx context.Context, fileID string) (bool, error) {
	// 查找文件
	paths, err := filepath.Glob(filepath.Join(d.basePath, "**", fileID+"*"))
	if err != nil {
		utils.Errorf("local exists: glob failed, fileID=%s, error=%v", fileID, err)
		return false, err
	}

	if len(paths) > 0 {
		return true, nil
	}

	// 如果没找到，尝试直接在根目录查找
	paths, err = filepath.Glob(filepath.Join(d.basePath, fileID+"*"))
	if err != nil {
		utils.Errorf("local exists: glob failed, fileID=%s, error=%v", fileID, err)
		return false, err
	}

	return len(paths) > 0, nil
}

// Stat 获取文件信息
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - *FileInfo: 文件信息
//   - error: 获取失败时的错误
func (d *LocalDriver) Stat(ctx context.Context, fileID string) (*FileInfo, error) {
	// 查找文件
	paths, err := filepath.Glob(filepath.Join(d.basePath, "**", fileID+"*"))
	if err != nil {
		utils.Errorf("local stat: glob failed, fileID=%s, error=%v", fileID, err)
		return nil, err
	}

	// 如果没找到，尝试直接在根目录查找
	if len(paths) == 0 {
		paths, err = filepath.Glob(filepath.Join(d.basePath, fileID+"*"))
		if err != nil {
			utils.Errorf("local stat: glob failed, fileID=%s, error=%v", fileID, err)
			return nil, err
		}
	}

	if len(paths) == 0 {
		utils.Warnf("local stat: file not found, fileID=%s", fileID)
		return nil, ErrFileNotFound
	}

	filePath := paths[0]
	stat, err := os.Stat(filePath)
	if err != nil {
		utils.Errorf("local stat: stat file failed, path=%s, error=%v", filePath, err)
		return nil, err
	}

	utils.Debugf("local stat: success, fileID=%s, path=%s", fileID, filePath)

	return &FileInfo{
		FileID:    fileID,
		Name:      filepath.Base(filePath),
		Size:      stat.Size(),
		MimeType:  getMimeType(filepath.Ext(filePath)),
		ChannelID: d.channelID,
		CreatedAt: stat.ModTime(),
	}, nil
}

// GetQuota 获取存储配额信息
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - *QuotaInfo: 配额信息
//   - error: 获取失败时的错误
func (d *LocalDriver) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	var totalSize int64
	var fileCount int

	// 遍历存储目录，统计文件数量和总大小
	err := filepath.Walk(d.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		utils.Errorf("local get quota: walk directory failed, error=%v", err)
		return nil, err
	}

	utils.Debugf("local get quota: totalSize=%d, fileCount=%d", totalSize, fileCount)

	return &QuotaInfo{
		UsedSpace:  totalSize,
		TotalSpace: 0, // 本地存储无限制
		FileCount:  fileCount,
	}, nil
}

// HealthCheck 检查本地存储健康状态
// 通过创建和删除测试文件来验证存储是否可写
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - error: 检查失败时的错误
func (d *LocalDriver) HealthCheck(ctx context.Context) error {
	testFile := filepath.Join(d.basePath, ".healthcheck")
	file, err := os.Create(testFile)
	if err != nil {
		utils.Errorf("local health check: create test file failed, error=%v", err)
		return err
	}
	file.Close()

	// 删除测试文件
	if err := os.Remove(testFile); err != nil {
		utils.Errorf("local health check: remove test file failed, error=%v", err)
		return err
	}

	return nil
}
