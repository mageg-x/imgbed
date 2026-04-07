package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/imgbed/server/utils"
)

// 注册HuggingFace存储驱动
func init() {
	RegisterDriver(StorageTypeHuggingFace, NewHuggingFaceDriver)
}

// HuggingFaceDriver HuggingFace存储驱动
// 实现 LFS 上传协议将文件上传到 HuggingFace Spaces 的数据集或模型仓库
type HuggingFaceDriver struct {
	token             string       // HuggingFace API Token
	repoID            string       // 仓库ID (username/repo-name)
	repoType          string       // 仓库类型 (dataset/model)
	client            *http.Client // HTTP客户端
	channelIDInternal string       // 内部通道ID
}

// HuggingFaceConfig HuggingFace存储配置
type HuggingFaceConfig struct {
	Token    string `json:"token"`    // API Token
	RepoID   string `json:"repoId"`   // 仓库ID
	RepoType string `json:"repoType"` // 仓库类型
}

// LFS Batch 响应结构
type lfsBatchResponse struct {
	Objects []lfsObject `json:"objects"`
}

type lfsObject struct {
	Oid     string       `json:"oid"`
	Size    int64        `json:"size"`
	Error   *lfsError   `json:"error,omitempty"`
	Actions *lfsActions  `json:"actions,omitempty"`
}

type lfsError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type lfsActions struct {
	Upload *lfsUploadAction `json:"upload,omitempty"`
}

type lfsUploadAction struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresAt string            `json:"expiresAt,omitempty"`
}

// Preupload 响应结构
type preuploadResponse struct {
	Files []preuploadFile `json:"files"`
}

type preuploadFile struct {
	Path       string `json:"path"`
	UploadMode string `json:"uploadMode"`
	Size       int64  `json:"size"`
}

// NewHuggingFaceDriver 创建HuggingFace存储驱动实例
func NewHuggingFaceDriver(cfg *ChannelConfig) (StorageDriver, error) {
	token, _ := cfg.Config["token"].(string)
	repoID, _ := cfg.Config["repoId"].(string)
	repoType, _ := cfg.Config["repoType"].(string)

	if repoType == "" {
		repoType = "dataset"
	}

	if token == "" || repoID == "" {
		utils.Errorf("new huggingface driver: token and repo id are required")
		return nil, fmt.Errorf("huggingface token and repo id are required")
	}

	utils.Infof("new huggingface driver: success, repoID=%s, repoType=%s", repoID, repoType)

	return &HuggingFaceDriver{
		token:             token,
		repoID:            repoID,
		repoType:          repoType,
		channelIDInternal: cfg.ID,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Name 返回驱动名称
func (d *HuggingFaceDriver) Name() string {
	return "HuggingFace"
}

// getAPIPath 返回 API 路径的复数形式
// HuggingFace API 使用复数形式：datasets, models, spaces
func (d *HuggingFaceDriver) getAPIPath() string {
	switch d.repoType {
	case "dataset":
		return "datasets"
	case "model":
		return "models"
	case "space":
		return "spaces"
	default:
		return d.repoType + "s"
	}
}

// Type 返回存储类型
func (d *HuggingFaceDriver) Type() StorageType {
	return StorageTypeHuggingFace
}

// Upload 上传文件到HuggingFace (使用LFS协议)
func (d *HuggingFaceDriver) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	// 读取文件内容
	data, err := io.ReadAll(req.Reader)
	if err != nil {
		utils.Errorf("huggingface upload: read file failed, error=%v", err)
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	fileSize := int64(len(data))

	// 生成文件ID
	fileID := req.FileID
	if fileID == "" {
		fileID = generateFileID()
	}

	// 添加文件扩展名
	ext := ""
	if idx := strings.LastIndex(req.FileName, "."); idx != -1 {
		ext = req.FileName[idx:]
	}
	fileIDWithExt := fileID + ext

	// 构建文件路径
	path := fileIDWithExt
	if req.Directory != "" {
		path = req.Directory + "/" + fileIDWithExt
	}

	// 计算 SHA256
	hash := sha256.Sum256(data)
	oid := hex.EncodeToString(hash[:])

	utils.Debugf("huggingface upload: fileSize=%d, oid=%s, path=%s", fileSize, oid, path)

	// 获取文件样本（前512字节的base64）
	sample := ""
	if fileSize > 512 {
		sample = base64.StdEncoding.EncodeToString(data[:512])
	} else {
		sample = base64.StdEncoding.EncodeToString(data)
	}

	// 步骤1: Preupload - 检查是否需要LFS
	needsLfs, err := d.preupload(path, fileSize, sample)
	if err != nil {
		utils.Errorf("huggingface upload: preupload failed, error=%v", err)
		return nil, fmt.Errorf("preupload failed: %w", err)
	}

	if !needsLfs {
		// 小文件直接提交
		err = d.commitDirectFile(path, data, "Upload "+fileIDWithExt)
		if err != nil {
			utils.Errorf("huggingface upload: direct commit failed, error=%v", err)
			return nil, fmt.Errorf("direct commit failed: %w", err)
		}
	} else {
		// 大文件使用LFS协议
		err = d.uploadWithLFS(path, oid, fileSize, data)
		if err != nil {
			utils.Errorf("huggingface upload: lfs upload failed, error=%v", err)
			return nil, fmt.Errorf("lfs upload failed: %w", err)
		}
	}

	// 构建访问URL
	fileURL := d.getFileURL(path)

	utils.Debugf("huggingface upload: success, fileID=%s, path=%s, url=%s", fileIDWithExt, path, fileURL)

	return &UploadResult{
		FileID:    fileIDWithExt,
		URL:       fileURL,
		Size:      fileSize,
		ChannelID: d.channelIDInternal,
	}, nil
}

// preupload 检查文件是否需要LFS
func (d *HuggingFaceDriver) preupload(path string, fileSize int64, sample string) (bool, error) {
	url := fmt.Sprintf("https://huggingface.co/api/%s/%s/preupload/main", d.getAPIPath(), d.repoID)

	reqBody := map[string]interface{}{
		"files": []map[string]interface{}{
			{
				"path":  path,
				"size":  fileSize,
				"sample": sample,
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("preupload failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var result preuploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	if len(result.Files) == 0 {
		return false, fmt.Errorf("preupload returned no files")
	}

	needsLfs := result.Files[0].UploadMode == "lfs"
	utils.Debugf("huggingface preupload: needsLfs=%v, uploadMode=%s", needsLfs, result.Files[0].UploadMode)

	return needsLfs, nil
}

// uploadWithLFS 使用LFS协议上传文件
func (d *HuggingFaceDriver) uploadWithLFS(path, oid string, fileSize int64, data []byte) error {
	// 步骤2: LFS Batch - 获取上传URL
	batchURL := fmt.Sprintf("https://huggingface.co/%s/%s.git/info/lfs/objects/batch", d.getAPIPath(), d.repoID)

	batchReq := map[string]interface{}{
		"operation": "upload",
		"transfers": []string{"basic", "multipart"},
		"hash_algo": "sha256",
		"ref": map[string]string{
			"name": "main",
		},
		"objects": []map[string]interface{}{
			{
				"oid":  oid,
				"size": fileSize,
			},
		},
	}

	body, err := json.Marshal(batchReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", batchURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lfs batch failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var batchResult lfsBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResult); err != nil {
		return err
	}

	if len(batchResult.Objects) == 0 {
		return fmt.Errorf("lfs batch returned no objects")
	}

	obj := batchResult.Objects[0]

	// 检查是否已存在
	if obj.Actions == nil || obj.Actions.Upload == nil {
		utils.Debugf("huggingface lfs: file already exists, oid=%s", oid)
		// 文件已存在，直接提交
	} else {
		// 步骤3: 上传到LFS存储
		err = d.uploadToLFS(obj.Actions.Upload, data, oid)
		if err != nil {
			return err
		}
	}

	// 步骤4: 提交LFS文件引用
	err = d.commitLfsFile(path, oid, fileSize, "Upload "+path)
	if err != nil {
		return err
	}

	return nil
}

// uploadToLFS 上传数据到LFS存储
func (d *HuggingFaceDriver) uploadToLFS(uploadAction *lfsUploadAction, data []byte, oid string) error {
	url := uploadAction.Href

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	// 设置必要的header
	for k, v := range uploadAction.Header {
		req.Header.Set(k, v)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lfs upload failed: %d - %s", resp.StatusCode, string(respBody))
	}

	utils.Debugf("huggingface lfs upload: success, oid=%s", oid)
	return nil
}

// commitLfsFile 提交LFS文件引用
func (d *HuggingFaceDriver) commitLfsFile(path, oid string, fileSize int64, commitMessage string) error {
	url := fmt.Sprintf("https://huggingface.co/api/%s/%s/commit/main", d.getAPIPath(), d.repoID)

	// NDJSON格式
	header := fmt.Sprintf(`{"key":"header","value":{"summary":"%s"}}`, commitMessage)
	lfsFile := fmt.Sprintf(`{"key":"lfsFile","value":{"path":"%s","algo":"sha256","size":%d,"oid":"%s"}}`, path, fileSize, oid)
	body := strings.NewReader(header + "\n" + lfsFile)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("commit failed: %d - %s", resp.StatusCode, string(respBody))
	}

	utils.Debugf("huggingface commit: success, path=%s", path)
	return nil
}

// commitDirectFile 直接提交文件（非LFS，用于小文本文件）
func (d *HuggingFaceDriver) commitDirectFile(path string, data []byte, commitMessage string) error {
	url := fmt.Sprintf("https://huggingface.co/api/%s/%s/commit/main", d.getAPIPath(), d.repoID)

	// 将数据转换为base64
	content := base64.StdEncoding.EncodeToString(data)

	header := fmt.Sprintf(`{"key":"header","value":{"summary":"%s"}}`, commitMessage)
	fileContent := fmt.Sprintf(`{"key":"file","value":{"path":"%s","content":"%s","encoding":"base64"}}`, path, content)
	body := strings.NewReader(header + "\n" + fileContent)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("direct commit failed: %d - %s", resp.StatusCode, string(respBody))
	}

	utils.Debugf("huggingface direct commit: success, path=%s", path)
	return nil
}

// Download 从HuggingFace下载文件
func (d *HuggingFaceDriver) Download(ctx context.Context, fileID string) (*DownloadResult, error) {
	url := d.getFileURL(fileID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		utils.Errorf("huggingface download: create request failed, error=%v", err)
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		utils.Errorf("huggingface download: send request failed, error=%v", err)
		return nil, fmt.Errorf("download failed: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		utils.Warnf("huggingface download: file not found, fileID=%s", fileID)
		return nil, ErrFileNotFound
	}

	utils.Debugf("huggingface download: success, fileID=%s", fileID)

	return &DownloadResult{
		Reader:   resp.Body,
		Size:     resp.ContentLength,
		MimeType: getMimeTypeFromKey(fileID),
	}, nil
}

// GetURL 获取文件的访问URL
func (d *HuggingFaceDriver) GetURL(ctx context.Context, fileID string) (string, error) {
	return d.getFileURL(fileID), nil
}

func (d *HuggingFaceDriver) getFileURL(path string) string {
	// resolve URL 格式：
	// - models: https://huggingface.co/{repo_id}/resolve/main/{path}  (repo_id = username/repo-name)
	// - datasets/spaces: https://huggingface.co/{api_path}/{repo_id}/resolve/main/{path}
	if d.repoType == "model" {
		return fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", d.repoID, path)
	}
	return fmt.Sprintf("https://huggingface.co/%s/%s/resolve/main/%s", d.getAPIPath(), d.repoID, path)
}

// Delete 删除HuggingFace文件
// HuggingFace 不支持直接的 DELETE API，需要通过 commit 来删除文件
func (d *HuggingFaceDriver) Delete(ctx context.Context, fileID string) error {
	url := fmt.Sprintf("https://huggingface.co/api/%s/%s/commit/main", d.getAPIPath(), d.repoID)

	// 构建 NDJSON 格式的删除请求
	header := `{"key":"header","value":{"summary":"Delete ` + fileID + `"}}`
	deletedFile := fmt.Sprintf(`{"key":"deletedFile","value":{"path":"%s"}}`, fileID)
	body := strings.NewReader(header + "\n" + deletedFile)

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		utils.Errorf("huggingface delete: create request failed, error=%v", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := d.client.Do(req)
	if err != nil {
		utils.Errorf("huggingface delete: send request failed, error=%v", err)
		return fmt.Errorf("delete failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		utils.Errorf("huggingface delete: failed, status=%d, body=%s", resp.StatusCode, string(respBody))
		return fmt.Errorf("delete failed: %s", string(respBody))
	}

	utils.Debugf("huggingface delete: success, fileID=%s", fileID)
	return nil
}

// Exists 检查文件是否存在
func (d *HuggingFaceDriver) Exists(ctx context.Context, fileID string) (bool, error) {
	url := fmt.Sprintf("https://huggingface.co/api/%s/%s/file/%s", d.getAPIPath(), d.repoID, fileID)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)

	resp, err := d.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// Stat 获取文件信息
func (d *HuggingFaceDriver) Stat(ctx context.Context, fileID string) (*FileInfo, error) {
	url := fmt.Sprintf("https://huggingface.co/api/%s/%s/file/%s", d.getAPIPath(), d.repoID, fileID)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrFileNotFound
	}

	return &FileInfo{
		FileID:    fileID,
		Size:      resp.ContentLength,
		MimeType:  getMimeTypeFromKey(fileID),
		ChannelID: d.channelIDInternal,
		CreatedAt: time.Now(),
	}, nil
}

// GetQuota 获取存储配额
func (d *HuggingFaceDriver) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	return &QuotaInfo{
		UsedSpace:  0,
		TotalSpace: 0,
		FileCount:  0,
	}, nil
}

// HealthCheck 检查仓库访问状态
func (d *HuggingFaceDriver) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("https://huggingface.co/api/%s/%s", d.getAPIPath(), d.repoID)
	utils.Debugf("huggingface health check: url=%s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		utils.Errorf("huggingface health check: create request failed, error=%v", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)

	resp, err := d.client.Do(req)
	if err != nil {
		utils.Errorf("huggingface health check: send request failed, error=%v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		utils.Errorf("huggingface health check: repo not accessible, status=%d", resp.StatusCode)
		return fmt.Errorf("huggingface repo not accessible, status=%d", resp.StatusCode)
	}

	return nil
}
