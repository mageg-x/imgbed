package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/storage"
	"github.com/imgbed/server/utils"
	"gorm.io/gorm"
)

const (
	AccessTypeUploadSuccess = "upload_success"
	AccessTypeUploadFailed  = "upload_failed"
)

// convertToCDNUrl 将原始直链转换为 CDN 代理地址
// 如果未启用 CDN 或 URL 不需要转换，则返回原始 URL
func convertToCDNUrl(originalUrl string) string {
	cdnConfig := config.GetCDNConfig()
	if !cdnConfig.Enabled || cdnConfig.ProxyUrl == "" || originalUrl == "" {
		return originalUrl
	}

	// 提取 URL 的基础路径（最后一个 / 之前）
	lastSlash := strings.LastIndex(originalUrl, "/")
	if lastSlash <= 0 {
		return originalUrl
	}

	baseUrl := originalUrl[:lastSlash]
	filePath := originalUrl[lastSlash+1:]

	// Base58 编码基础 URL
	encoded := utils.Base58Encode(baseUrl)

	// 拼接 CDN URL
	proxyUrl := strings.TrimSuffix(cdnConfig.ProxyUrl, "/")
	return fmt.Sprintf("%s/%s/%s", proxyUrl, encoded, filePath)
}

// GetCDNUrl 公开的 CDN URL 转换方法，供 handler 层使用
func (s *FileService) GetCDNUrl(originalUrl string, channelType string) string {
	if channelType == "local" {
		return originalUrl
	}
	return convertToCDNUrl(originalUrl)
}

// parseSearchSource 解析搜索字符串，支持 "source:文件名" 格式
// source 支持前缀模糊匹配（如 api_git 匹配 api_github、api_gitlab）
// 文件名支持模糊匹配
// 例如：api_git:logo → source LIKE 'api_git%' AND name LIKE '%logo%'
func parseSearchSource(search string) (sourcePattern, namePattern string) {
	if search == "" {
		return "", ""
	}
	idx := strings.Index(search, ":")
	if idx <= 0 || idx >= len(search)-1 {
		// 没有冒号或格式不对，整个作为文件名模糊搜索
		return "", search
	}
	sourcePart := search[:idx]
	namePart := search[idx+1:]

	// source 部分：如果包含下划线或恰好是已知类型，则作为前缀模糊匹配
	// 例如 api_git → LIKE 'api_git%'（匹配 api_github, api_gitlab 等）
	// 例如 admin → LIKE 'admin%'
	if sourcePart == "user" || sourcePart == "admin" || sourcePart == "anonymous" {
		return sourcePart, namePart
	}
	// 其他情况（如 api_git），按前缀匹配
	return sourcePart, namePart
}

// fts5Search 使用 FTS5 全文搜索查找匹配的文件ID
// keyword 支持中文分词和前缀匹配
// 返回匹配的文件ID列表，如果 FTS5 不可用返回 nil
func fts5Search(keyword string) ([]string, error) {
	if keyword == "" {
		return nil, nil
	}
	// 检查 FTS5 表是否存在
	var count int64
	database.DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='files_fts'").Scan(&count)
	if count == 0 {
		return nil, nil // FTS5 不可用
	}

	var ids []string
	matchClause := keyword + "*"
	err := database.DB.Table("files_fts").
		Select("id").
		Where("files_fts MATCH ?", matchClause).
		Pluck("id", &ids).Error
	if err != nil {
		utils.Warnf("fts5 search failed, keyword=%s, error=%v", keyword, err)
		return nil, err
	}
	return ids, nil
}

type FileService struct {
	channelService    *ChannelService
	moderationService *ModerationService
}

func NewFileService() *FileService {
	return &FileService{
		channelService:    NewChannelService(),
		moderationService: NewModerationService(),
	}
}

func (s *FileService) recordFileAccess(fileID string, accessType string, ip string, userAgent string) {
	access := &model.FileAccess{
		FileID:     fileID,
		IP:         ip,
		UserAgent:  userAgent,
		AccessAt:   time.Now(),
		AccessType: accessType,
	}
	if err := database.DB.Create(access).Error; err != nil {
		utils.Warnf("record file access: failed, fileID=%s, type=%s, error=%v", fileID, accessType, err)
	}
}

// CheckFileByChecksum 检查文件是否已存在（用于秒传）
// 参数：
//   - checksum: 文件 SHA256 哈希
//
// 返回：
//   - *model.File: 已存在的文件记录，不存在则返回 nil
//   - error: 查询错误
func (s *FileService) CheckFileByChecksum(checksum string) (*model.File, error) {
	if checksum == "" {
		return nil, nil
	}

	var file model.File
	err := database.DB.Where("checksum = ?", checksum).First(&file).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		utils.Errorf("check file by checksum: query failed, checksum=%s, error=%v", checksum, err)
		return nil, err
	}

	utils.Debugf("check file by checksum: found, checksum=%s, fileID=%s", checksum, file.ID)
	return &file, nil
}

// Upload 上传单个文件
func (s *FileService) Upload(ctx context.Context, file *multipart.FileHeader, channelID string, tags []string, source string, uploadedByToken string, clientIP string, userAgent string) (*model.UploadResult, error) {
	allowedTypes := config.GetStringSlice("upload.allowed_types")

	actualMimeType, err := utils.ValidateFileForUpload(file, allowedTypes)
	if err != nil {
		utils.Warnf("upload: file type validation failed, filename=%s, error=%v", file.Filename, err)
		return nil, fmt.Errorf("file type not allowed: %w", err)
	}

	safeFilename := utils.SanitizeFilename(file.Filename)

	reader, err := file.Open()
	if err != nil {
		utils.Errorf("upload: open file failed, filename=%s, error=%v", safeFilename, err)
		return nil, fmt.Errorf("open file failed: %w", err)
	}
	defer reader.Close()

	// 读取文件数据到内存
	fileData, err := io.ReadAll(reader)
	if err != nil {
		utils.Errorf("upload: read file failed, filename=%s, error=%v", safeFilename, err)
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	// 计算原始文件的 SHA256 校验和（用于秒传和去重）
	fileChecksum := utils.CalcSHA256(fileData)

	fileID := utils.GenerateID()
	mimeType := actualMimeType

	// SVG 安全清理
	if mimeType == "image/svg+xml" {
		cleanedData, err := utils.SanitizeSVG(fileData)
		if err != nil {
			utils.Warnf("upload: svg sanitization warning, filename=%s, error=%v", safeFilename, err)
		} else {
			fileData = cleanedData
		}
	}

	// 图片压缩处理
	var compressedData []byte
	var outputMimeType string

	if utils.IsImageFile(file) && config.GetBool("compression.enabled") {
		compressionConfig := utils.CompressionConfig{
			Enabled:   true,
			Quality:   config.GetInt("compression.quality"),
			Format:    config.GetString("compression.format"),
			MaxWidth:  config.GetInt("compression.max_width"),
			MaxHeight: config.GetInt("compression.max_height"),
		}

		compressedData, outputMimeType, err = utils.CompressImage(file, bytes.NewReader(fileData), compressionConfig)
		if err != nil {
			utils.Warnf("upload: compression failed, filename=%s, error=%v, using original", safeFilename, err)
			// 压缩失败，使用原始文件
			compressedData = fileData
			outputMimeType = mimeType
		} else {
			mimeType = outputMimeType
			utils.Infof("upload: image compressed, filename=%s, originalSize=%d, compressedSize=%d",
				safeFilename, file.Size, len(compressedData))
		}
	} else {
		compressedData = fileData
		outputMimeType = mimeType
	}

	// 内容审核（仅对图片进行审核）
	if s.moderationService.ShouldModerate(mimeType) {
		// 执行内容审核
		moderationResult, moderationErr := s.moderationService.CheckContent(ctx, compressedData, safeFilename)
		if moderationErr != nil {
			utils.Warnf("upload: content moderation failed, filename=%s, error=%v", safeFilename, moderationErr)
			// 审核失败不阻止上传，但记录警告
		} else if !moderationResult.Safe {
			utils.Warnf("upload: content moderation rejected, filename=%s, labels=%v, action=%s",
				safeFilename, moderationResult.Labels, moderationResult.SuggestedAction)
			// 根据配置决定是否拒绝上传
			if moderationResult.SuggestedAction == "reject" {
				return nil, fmt.Errorf("content moderation rejected: inappropriate content detected")
			}
		} else {
			utils.Infof("upload: content moderation passed, filename=%s, provider=%s", safeFilename, moderationResult.Provider)
		}
	}

	if channelID == "" {
		channelID, err = s.channelService.SelectChannel(ctx, file.Size)
		if err != nil {
			utils.Errorf("upload: select channel failed, filename=%s, size=%d, error=%v", safeFilename, file.Size, err)
			return nil, fmt.Errorf("select channel failed: %w", err)
		}
	}

	driver, err := s.channelService.GetDriver(channelID)
	if err != nil {
		utils.Errorf("upload: get driver failed, channelID=%s, filename=%s, error=%v", channelID, safeFilename, err)
		return nil, fmt.Errorf("get driver failed: %w", err)
	}

	// 使用压缩后的数据或原始数据
	uploadReader := bytes.NewReader(compressedData)
	uploadSize := int64(len(compressedData))

	uploadReq := &storage.UploadRequest{
		FileID:    fileID,
		FileName:  safeFilename,
		FileSize:  uploadSize,
		Reader:    uploadReader,
		Tags:      tags,
		ChannelID: channelID,
	}

	result, err := driver.Upload(ctx, uploadReq)
	if err != nil {
		utils.Errorf("upload: driver upload failed, channelID=%s, filename=%s, error=%v", channelID, safeFilename, err)
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	// 使用驱动返回的 fileID（包含扩展名）
	fileID = result.FileID

	fileRecord := &model.File{
		ID:              fileID,
		Name:            safeFilename,
		OriginalName:    file.Filename,
		Size:            result.Size,
		Type:            mimeType,
		ChannelID:       channelID,
		ChannelType:     string(driver.Type()),
		Tags:            strings.Join(tags, ","),
		URL:             result.URL,
		Checksum:        fileChecksum,
		Source:          source,
		UploadedByToken: uploadedByToken,
	}

	if err := database.DB.Create(fileRecord).Error; err != nil {
		utils.Errorf("upload: save file record failed, fileID=%s, filename=%s, error=%v", fileID, safeFilename, err)
		return nil, fmt.Errorf("save file record failed: %w", err)
	}

	s.channelService.UpdateUsage(channelID, result.Size)

	s.recordFileAccess(fileID, AccessTypeUploadSuccess, clientIP, userAgent)

	utils.Infof("upload: success, fileID=%s, filename=%s, channelID=%s, size=%d", fileID, file.Filename, channelID, result.Size)

	// 如果驱动返回的URL为空，使用本地代理URL
	fileURL := result.URL
	if fileURL == "" {
		fileURL = "/api/v1/file/" + fileID
	}

	// CDN 转换（local 渠道不转换）
	cdnURL := fileURL
	if string(driver.Type()) != "local" {
		cdnURL = convertToCDNUrl(fileURL)
	}

	// 构建多格式链接
	links := model.Links{
		URL:      cdnURL,
		Markdown: fmt.Sprintf("![%s](%s)", file.Filename, cdnURL),
		HTML:     fmt.Sprintf(`<img src="%s" alt="%s">`, cdnURL, file.Filename),
	}

	return &model.UploadResult{
		ID:          fileID,
		Name:        file.Filename,
		URL:         cdnURL,
		Size:        result.Size,
		Type:        mimeType,
		Channel:     channelID,
		ChannelType: string(driver.Type()),
		Tags:        tags,
		UploadedAt:  time.Now().Unix(),
		Links:       links,
	}, nil
}

// UploadWithRetry 带重试的上传操作
// 支持排除已失败的渠道，实现真正的渠道切换
func (s *FileService) UploadWithRetry(ctx context.Context, file *multipart.FileHeader, tags []string, source string, uploadedByToken string, retryCount int, clientIP string, userAgent string) (*model.UploadResult, error) {
	var lastErr error
	failedChannels := make(map[string]bool)
	consecutiveFailures := make(map[string]int)

	for i := 0; i <= retryCount; i++ {
		var channelID string
		var err error
		// 第一次选择使用策略（round_robin/random/priority），重试时排除已失败的渠道
		if i == 0 {
			channelID, err = s.channelService.SelectChannel(ctx, file.Size)
		} else {
			channelID, err = s.channelService.SelectChannelExcluding(ctx, file.Size, failedChannels)
		}
		if err != nil {
			utils.Warnf("upload with retry: no available channels, retry=%d/%d, filename=%s, error=%v", i, retryCount, file.Filename, err)
			break
		}

		result, err := s.Upload(ctx, file, channelID, tags, source, uploadedByToken, clientIP, userAgent)
		if err == nil {
			return result, nil
		}

		lastErr = err
		failedChannels[channelID] = true
		consecutiveFailures[channelID]++

		if consecutiveFailures[channelID] >= 3 {
			if markErr := s.channelService.MarkChannelCooldown(ctx, channelID); markErr != nil {
				utils.Warnf("upload with retry: mark cooldown failed, channelID=%s, error=%v", channelID, markErr)
			}
		}

		if strings.Contains(err.Error(), "quota") || strings.Contains(err.Error(), "limit") ||
			strings.Contains(err.Error(), "disabled") || strings.Contains(err.Error(), "not found") {
			utils.Warnf("upload with retry: channel error, retry=%d/%d, channelID=%s, filename=%s, error=%v",
				i, retryCount, channelID, file.Filename, err)
			continue
		}

		utils.Warnf("upload with retry: upload failed, retry=%d/%d, channelID=%s, filename=%s, error=%v",
			i, retryCount, channelID, file.Filename, err)
	}

	s.recordFileAccess("", AccessTypeUploadFailed, clientIP, userAgent)

	utils.Errorf("upload with retry: failed after %d retries, filename=%s, error=%v", retryCount, file.Filename, lastErr)
	return nil, fmt.Errorf("upload failed after %d retries: %w", retryCount, lastErr)
}

// Download 下载文件
func (s *FileService) Download(ctx context.Context, fileID string) (io.ReadCloser, string, int64, error) {
	var file model.File
	if err := database.DB.Where("id = ?", fileID).First(&file).Error; err != nil {
		utils.Errorf("download: file not found, fileID=%s, error=%v", fileID, err)
		return nil, "", 0, fmt.Errorf("file not found")
	}

	driver, err := s.channelService.GetDriver(file.ChannelID)
	if err != nil {
		utils.Errorf("download: get driver failed, fileID=%s, channelID=%s, error=%v", fileID, file.ChannelID, err)
		return nil, "", 0, fmt.Errorf("get driver failed: %w", err)
	}

	result, err := driver.Download(ctx, fileID)
	if err != nil {
		utils.Errorf("download: driver download failed, fileID=%s, error=%v", fileID, err)
		return nil, "", 0, fmt.Errorf("download failed: %w", err)
	}

	if err := database.DB.Model(&file).Update("access_count", file.AccessCount+1).Error; err != nil {
		utils.Warnf("download: update access count failed, fileID=%s, error=%v", fileID, err)
	}

	return result.Reader, result.MimeType, result.Size, nil
}

// Delete 删除单个文件
// 参数：
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回：
//   - error: 错误信息
func (s *FileService) Delete(ctx context.Context, fileID string, anonymousID string) error {
	// 查找文件记录
	var file model.File
	if err := database.DB.Where("id = ?", fileID).First(&file).Error; err != nil {
		utils.Errorf("delete: file not found, fileID=%s, error=%v", fileID, err)
		return fmt.Errorf("file not found")
	}

	// 如果文件有 UploadedByToken（匿名上传），验证 anonymousID 是否匹配
	if anonymousID != "" && file.UploadedByToken != "" {
		if anonymousID != file.UploadedByToken {
			utils.Warnf("delete: anonymous id mismatch, fileID=%s", fileID)
			return fmt.Errorf("authorization required")
		}
	}

	// 获取存储驱动
	driver, err := s.channelService.GetDriver(file.ChannelID)
	if err != nil {
		// 获取驱动失败，记录警告但继续删除数据库记录
		utils.Warnf("delete: get driver failed, fileID=%s, channelID=%s, error=%v", fileID, file.ChannelID, err)
	} else {
		// 删除存储中的文件
		if deleteErr := driver.Delete(ctx, fileID); deleteErr != nil {
			utils.Warnf("delete: driver delete failed, fileID=%s, error=%v", fileID, deleteErr)
		}
	}

	// 删除数据库记录
	if err := database.DB.Delete(&file).Error; err != nil {
		utils.Errorf("delete: delete file record failed, fileID=%s, error=%v", fileID, err)
		return fmt.Errorf("delete file record failed: %w", err)
	}

	// 更新渠道使用量
	s.channelService.UpdateUsage(file.ChannelID, -file.Size)

	utils.Infof("delete: success, fileID=%s, channelID=%s, size=%d", fileID, file.ChannelID, file.Size)
	return nil
}

// DeleteMultiple 批量删除文件（使用事务保护数据一致性）
// 参数：
//   - ctx: 上下文
//   - fileIDs: 要删除的文件ID列表
//
// 返回：
//   - []string: 成功删除的文件ID列表
//   - []string: 删除失败的文件ID列表
//   - error: 事务错误（如果事务失败）
func (s *FileService) DeleteMultiple(ctx context.Context, fileIDs []string) ([]string, []string, error) {
	if len(fileIDs) == 0 {
		return nil, nil, nil
	}

	successIDs := make([]string, 0)
	failedIDs := make([]string, 0)
	filesToDelete := make([]model.File, 0)

	for _, id := range fileIDs {
		var file model.File
		if err := database.DB.Where("id = ?", id).First(&file).Error; err != nil {
			utils.Warnf("delete multiple: file not found, fileID=%s", id)
			failedIDs = append(failedIDs, id)
			continue
		}
		filesToDelete = append(filesToDelete, file)
	}

	if len(filesToDelete) == 0 {
		return successIDs, failedIDs, nil
	}

	fileIDsToDelete := make([]string, len(filesToDelete))
	channelSizeMap := make(map[string]int64)
	for i, f := range filesToDelete {
		fileIDsToDelete[i] = f.ID
		channelSizeMap[f.ChannelID] += f.Size
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id IN ?", fileIDsToDelete).Delete(&model.File{}).Error; err != nil {
			utils.Errorf("delete multiple: batch delete files failed, error=%v", err)
			return err
		}

		if err := tx.Where("file_id IN ?", fileIDsToDelete).Delete(&model.FileAccess{}).Error; err != nil {
			utils.Warnf("delete multiple: batch delete access records failed, error=%v", err)
		}

		return nil
	})

	if err != nil {
		utils.Errorf("delete multiple: transaction failed, error=%v", err)
		return successIDs, fileIDs, err
	}

	for chID, size := range channelSizeMap {
		s.channelService.UpdateUsage(chID, -size)
	}

	go func() {
		for _, f := range filesToDelete {
			driver, err := s.channelService.GetDriver(f.ChannelID)
			if err != nil {
				utils.Warnf("delete multiple async: get driver failed, fileID=%s, error=%v", f.ID, err)
				continue
			}
			if driver != nil {
				if deleteErr := driver.Delete(context.Background(), f.ID); deleteErr != nil {
					utils.Warnf("delete multiple async: driver delete failed, fileID=%s, error=%v", f.ID, deleteErr)
				}
			}
		}
	}()

	return fileIDsToDelete, failedIDs, nil
}

// List 获取文件列表
// 参数：
//   - ctx: 上下文
//   - page: 页码
//   - pageSize: 每页数量
//   - search: 搜索关键字
//   - startTime: 开始时间（Unix时间戳，筛选在此时间之后上传的文件）
//   - endTime: 结束时间（Unix时间戳，筛选在此时间之前上传的文件）
//   - olderThan: N天前的文件（会覆盖startTime/endTime）
//
// 返回：
//   - []model.FileInfo: 文件信息列表
//   - int64: 总数
//   - error: 错误信息
func (s *FileService) List(ctx context.Context, page, pageSize int, search string, source string, startTime, endTime, olderThan int64) ([]model.FileInfo, int64, error) {
	var files []model.File
	var total int64

	query := database.DB.Model(&model.File{})

	// 搜索条件：支持 "source:文件名" 格式，source 支持前缀模糊匹配
	searchSource, searchName := parseSearchSource(search)
	if searchSource != "" {
		query = query.Where("source LIKE ?", searchSource+"%")
	}

	// 文件名搜索：优先使用 FTS5 加速，失败则 fallback 到 LIKE
	if searchName != "" {
		ids, err := fts5Search(searchName)
		if err == nil && ids != nil && len(ids) > 0 {
			query = query.Where("id IN (?)", ids)
		} else if err != nil || ids == nil {
			// FTS5 不可用或失败，fallback 到 LIKE
			query = query.Where("name LIKE ? OR original_name LIKE ?", "%"+searchName+"%", "%"+searchName+"%")
		}
	}

	// 来源筛选（优先级高于 search 解析的 source）
	if source != "" {
		query = query.Where("source = ?", source)
	}

	// 时间范围筛选
	if olderThan > 0 {
		// olderThan 表示 N 天前的文件
		cutoff := time.Now().AddDate(0, 0, -int(olderThan))
		query = query.Where("created_at < ?", cutoff)
	} else {
		// startTime 和 endTime 范围筛选
		if startTime > 0 {
			query = query.Where("created_at >= ?", time.Unix(startTime, 0))
		}
		if endTime > 0 {
			query = query.Where("created_at <= ?", time.Unix(endTime, 0))
		}
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		utils.Errorf("list: query failed, page=%d, pageSize=%d, error=%v", page, pageSize, err)
		return nil, 0, err
	}

	result := make([]model.FileInfo, len(files))
	for i, f := range files {
		// 原始 URL 直接从数据库取
		originalURL := f.URL
		// local 渠道不进行 CDN 转换
		cdnURL := originalURL
		if f.ChannelType != "local" {
			cdnURL = convertToCDNUrl(originalURL)
		}
		links := model.Links{
			URL:      cdnURL,
			Markdown: fmt.Sprintf("![%s](%s)", f.Name, cdnURL),
			HTML:     fmt.Sprintf(`<img src="%s" alt="%s">`, cdnURL, f.Name),
		}
		result[i] = model.FileInfo{
			ID:          f.ID,
			Name:        f.Name,
			URL:         cdnURL,
			OriginalURL: originalURL,
			Size:        f.Size,
			Type:        f.Type,
			Channel:     f.ChannelID,
			ChannelType: f.ChannelType,
			Directory:   f.Directory,
			Tags:        utils.ParseTags(f.Tags),
			UploadedAt:  f.CreatedAt.Unix(),
			AccessCount: f.AccessCount,
			Source:      f.Source,
			Links:       links,
		}
	}

	return result, total, nil
}

func (s *FileService) ListIds(ctx context.Context, search string, source string, startTime, endTime, olderThan int64) ([]string, int64, error) {
	var ids []string
	var total int64

	query := database.DB.Model(&model.File{})

	// 搜索条件：支持 "source:文件名" 格式，source 支持前缀模糊匹配
	searchSource, searchName := parseSearchSource(search)
	if searchSource != "" {
		query = query.Where("source LIKE ?", searchSource+"%")
	}

	// 文件名搜索：优先使用 FTS5 加速，失败则 fallback 到 LIKE
	if searchName != "" {
		ids, err := fts5Search(searchName)
		if err == nil && ids != nil && len(ids) > 0 {
			query = query.Where("id IN (?)", ids)
		} else if err != nil || ids == nil {
			// FTS5 不可用或失败，fallback 到 LIKE
			query = query.Where("name LIKE ? OR original_name LIKE ?", "%"+searchName+"%", "%"+searchName+"%")
		}
	}

	// 来源筛选（优先级高于 search 解析的 source）
	if source != "" {
		query = query.Where("source = ?", source)
	}

	if olderThan > 0 {
		cutoff := time.Now().AddDate(0, 0, -int(olderThan))
		query = query.Where("created_at < ?", cutoff)
	} else {
		if startTime > 0 {
			query = query.Where("created_at >= ?", time.Unix(startTime, 0))
		}
		if endTime > 0 {
			query = query.Where("created_at <= ?", time.Unix(endTime, 0))
		}
	}

	query.Count(&total)

	if err := query.Pluck("id", &ids).Error; err != nil {
		utils.Errorf("list ids: query failed, error=%v", err)
		return nil, 0, err
	}

	return ids, total, nil
}

// 参数：
//   - ctx: 上下文
//   - olderThan: N天前的文件（0表示所有文件）
//   - startTime: 开始时间（Unix时间戳）
//   - endTime: 结束时间（Unix时间戳）
//   - channelID: 指定渠道ID（空表示所有渠道）
//
// 返回：
//   - *CleanupPreview: 预览结果
//   - error: 错误信息
func (s *FileService) CleanupPreview(ctx context.Context, olderThan, startTime, endTime int64, channelID string) (*CleanupPreview, error) {
	query := database.DB.Model(&model.File{})

	// 构建筛选条件
	if olderThan > 0 {
		cutoff := time.Now().AddDate(0, 0, -int(olderThan))
		query = query.Where("created_at < ?", cutoff)
	} else {
		if startTime > 0 {
			query = query.Where("created_at >= ?", time.Unix(startTime, 0))
		}
		if endTime > 0 {
			query = query.Where("created_at <= ?", time.Unix(endTime, 0))
		}
	}

	if channelID != "" {
		query = query.Where("channel_id = ?", channelID)
	}

	// 统计将要删除的文件数量和大小
	var total int64
	var totalSize int64
	query.Count(&total)

	// 获取总大小
	type result struct {
		Size int64
	}
	var sizes []result
	query.Select("COALESCE(SUM(size), 0) as size").Find(&sizes)
	if len(sizes) > 0 {
		totalSize = sizes[0].Size
	}

	// 获取前10个预览
	var previewFiles []model.File
	query.Order("created_at ASC").Limit(10).Find(&previewFiles)

	previews := make([]FilePreview, len(previewFiles))
	for i, f := range previewFiles {
		previews[i] = FilePreview{
			ID:         f.ID,
			Name:       f.Name,
			Size:       f.Size,
			UploadedAt: f.CreatedAt.Unix(),
			Channel:    f.ChannelID,
		}
	}

	utils.Infof("cleanup preview: count=%d, totalSize=%d", total, totalSize)

	return &CleanupPreview{
		Count:     total,
		TotalSize: totalSize,
		Preview:   previews,
	}, nil
}

// Cleanup 执行清理旧文件
// 参数：
//   - ctx: 上下文
//   - olderThan: N天前的文件（0表示所有文件）
//   - startTime: 开始时间（Unix时间戳）
//   - endTime: 结束时间（Unix时间戳）
//   - channelID: 指定渠道ID（空表示所有渠道）
//
// 返回：
//   - *CleanupResult: 清理结果
//   - error: 错误信息
func (s *FileService) Cleanup(ctx context.Context, olderThan, startTime, endTime int64, channelID string) (*CleanupResult, error) {
	query := database.DB.Model(&model.File{})

	if olderThan > 0 {
		cutoff := time.Now().AddDate(0, 0, -int(olderThan))
		query = query.Where("created_at < ?", cutoff)
	} else {
		if startTime > 0 {
			query = query.Where("created_at >= ?", time.Unix(startTime, 0))
		}
		if endTime > 0 {
			query = query.Where("created_at <= ?", time.Unix(endTime, 0))
		}
	}

	if channelID != "" {
		query = query.Where("channel_id = ?", channelID)
	}

	var files []model.File
	if err := query.Find(&files).Error; err != nil {
		utils.Errorf("cleanup: query files failed, error=%v", err)
		return nil, err
	}

	if len(files) == 0 {
		return &CleanupResult{}, nil
	}

	fileIDs := make([]string, len(files))
	fileSizeMap := make(map[string]int64)
	channelSizeMap := make(map[string]int64)

	for i, f := range files {
		fileIDs[i] = f.ID
		fileSizeMap[f.ID] = f.Size
		channelSizeMap[f.ChannelID] += f.Size
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		utils.Errorf("cleanup: begin transaction failed, error=%v", tx.Error)
		return nil, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("id IN ?", fileIDs).Delete(&model.File{}).Error; err != nil {
		utils.Errorf("cleanup: batch delete files failed, error=%v", err)
		tx.Rollback()
		return nil, err
	}

	if err := tx.Where("file_id IN ?", fileIDs).Delete(&model.FileAccess{}).Error; err != nil {
		utils.Warnf("cleanup: batch delete access records failed, error=%v", err)
	}

	if err := tx.Commit().Error; err != nil {
		utils.Errorf("cleanup: commit transaction failed, error=%v", err)
		return nil, err
	}

	go func() {
		for _, f := range files {
			driver, err := s.channelService.GetDriver(f.ChannelID)
			if err != nil {
				utils.Warnf("cleanup async: get driver failed, fileID=%s, error=%v", f.ID, err)
				continue
			}
			if driver != nil {
				if deleteErr := driver.Delete(context.Background(), f.ID); deleteErr != nil {
					utils.Warnf("cleanup async: driver delete failed, fileID=%s, error=%v", f.ID, deleteErr)
				}
			}
		}
	}()

	for chID, size := range channelSizeMap {
		s.channelService.UpdateUsage(chID, -size)
	}

	var totalSize int64
	for _, size := range fileSizeMap {
		totalSize += size
	}

	utils.Infof("cleanup: deletedCount=%d, freedSize=%d", len(files), totalSize)

	return &CleanupResult{
		DeletedCount: int64(len(files)),
		FailedCount:  0,
		FreedSize:    totalSize,
		FailedIDs:    []string{},
	}, nil
}

// CleanupPreview 清理预览结果
type CleanupPreview struct {
	Count     int64         `json:"count"`     // 将要删除的文件数量
	TotalSize int64         `json:"totalSize"` // 将要释放的空间（字节）
	Preview   []FilePreview `json:"preview"`   // 前10个预览
}

// FilePreview 文件预览信息
type FilePreview struct {
	ID         string `json:"id"`         // 文件ID
	Name       string `json:"name"`       // 文件名
	Size       int64  `json:"size"`       // 文件大小
	UploadedAt int64  `json:"uploadedAt"` // 上传时间
	Channel    string `json:"channel"`    // 存储渠道
}

// CleanupResult 清理结果
type CleanupResult struct {
	DeletedCount int64    `json:"deletedCount"` // 成功删除数量
	FailedCount  int64    `json:"failedCount"`  // 失败数量
	FreedSize    int64    `json:"freedSize"`    // 释放空间（字节）
	FailedIDs    []string `json:"failedIds"`    // 失败的ID列表
}

// GetInfo 获取文件详细信息
func (s *FileService) GetInfo(ctx context.Context, fileID string) (*model.FileInfo, error) {
	var file model.File
	if err := database.DB.Where("id = ?", fileID).First(&file).Error; err != nil {
		utils.Errorf("get info: file not found, fileID=%s, error=%v", fileID, err)
		return nil, fmt.Errorf("file not found")
	}

	// 原始 URL 直接从数据库取，CDN 转换后返回
	originalURL := file.URL
	// local 渠道不进行 CDN 转换
	cdnURL := originalURL
	if file.ChannelType != "local" {
		cdnURL = convertToCDNUrl(originalURL)
	}

	links := model.Links{
		URL:      cdnURL,
		Markdown: fmt.Sprintf("![%s](%s)", file.Name, cdnURL),
		HTML:     fmt.Sprintf(`<img src="%s" alt="%s">`, cdnURL, file.Name),
	}

	var lastAccessAt int64
	var lastAccess model.FileAccess
	if err := database.DB.Where("file_id = ? AND access_type = ?", file.ID, "view").Order("access_at DESC").First(&lastAccess).Error; err == nil {
		lastAccessAt = lastAccess.AccessAt.Unix()
	}

	info := &model.FileInfo{
		ID:           file.ID,
		Name:         file.Name,
		URL:          cdnURL,
		OriginalURL:  originalURL,
		Size:         file.Size,
		Type:         file.Type,
		Channel:      file.ChannelID,
		ChannelType:  file.ChannelType,
		Directory:    file.Directory,
		Tags:         utils.ParseTags(file.Tags),
		UploadedAt:   file.CreatedAt.Unix(),
		AccessCount:  file.AccessCount,
		LastAccessAt: lastAccessAt,
		Checksum:     file.Checksum,
		Source:       file.Source,
		Links:        links,
	}

	return info, nil
}

// GetURL 获取文件访问URL
func (s *FileService) GetURL(ctx context.Context, fileID string) (string, error) {
	var file model.File
	if err := database.DB.Where("id = ?", fileID).First(&file).Error; err != nil {
		utils.Errorf("get url: file not found, fileID=%s, error=%v", fileID, err)
		return "", fmt.Errorf("file not found")
	}

	driver, err := s.channelService.GetDriver(file.ChannelID)
	if err != nil {
		utils.Errorf("get url: get driver failed, fileID=%s, channelID=%s, error=%v", fileID, file.ChannelID, err)
		return "", fmt.Errorf("get driver failed: %w", err)
	}

	url, err := driver.GetURL(ctx, fileID)
	if err != nil {
		return "", err
	}
	// CDN 转换
	return convertToCDNUrl(url), nil
}

func (s *FileService) GetUploadCount(ip string, date string) (int, error) {
	var count int64
	startOfDay, _ := time.Parse("2006-01-02", date)
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := database.DB.Model(&model.File{}).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Where("source = ? OR source = ?", "user", "anonymous").
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
