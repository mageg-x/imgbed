package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/utils"
)

type FileHandler struct {
	fileService *service.FileService
}

func NewFileHandler() *FileHandler {
	return &FileHandler{
		fileService: service.NewFileService(),
	}
}

// getSourceFromContext 从请求上下文获取上传来源
func (h *FileHandler) getSourceFromContext(c *gin.Context) string {
	authType, _ := c.Get("authType")
	role, _ := c.Get("role")

	if authType == "token" {
		// API Token 上传，来源是 api_xxx
		if apiToken, ok := c.Get("apiToken"); ok {
			if token, ok := apiToken.(*service.APIToken); ok {
				return "api_" + token.Name
			}
		}
		return "api"
	}

	// JWT 上传
	if role == "admin" {
		return "admin"
	}
	return "user"
}

func (h *FileHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.Warnf("upload: file is required, error=%v", err)
		response.ValidationError(c, "file is required")
		return
	}

	tags := c.PostFormArray("tags")

	retryCount := 3
	if rc := c.GetHeader("X-Retry-Count"); rc != "" {
		if val, err := strconv.Atoi(rc); err == nil {
			retryCount = val
		}
	}

	// 确定来源
	source := h.getSourceFromContext(c)

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	result, err := h.fileService.UploadWithRetry(c.Request.Context(), file, tags, source, "", retryCount, clientIP, userAgent)
	if err != nil {
		utils.Errorf("upload: upload failed, filename=%s, error=%v", file.Filename, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("upload: success, fileID=%s, filename=%s", result.ID, result.Name)
	response.Success(c, result)
}

// UploadMultiple 处理多文件上传
// POST /api/v1/upload/multiple
// 支持同时上传多个文件，逐个处理，返回每个文件的上传结果
func (h *FileHandler) UploadMultiple(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		utils.Warnf("upload multiple: invalid form data, error=%v", err)
		response.ValidationError(c, "invalid form data")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		utils.Warnf("upload multiple: no files uploaded")
		response.ValidationError(c, "no files uploaded")
		return
	}

	tags := c.PostFormArray("tags")
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	source := h.getSourceFromContext(c)

	results := make([]interface{}, 0)
	for _, file := range files {
		result, err := h.fileService.UploadWithRetry(c.Request.Context(), file, tags, source, "", 3, clientIP, userAgent)
		if err != nil {
			utils.Errorf("upload multiple: one file failed, filename=%s, error=%v", file.Filename, err)
			results = append(results, gin.H{
				"name":  file.Filename,
				"error": err.Error(),
			})
		} else {
			results = append(results, result)
		}
	}

	response.Success(c, results)
}

// Download 处理文件下载
// GET /api/v1/file/:id/download
// 通过文件ID获取文件内容并触发下载
func (h *FileHandler) Download(c *gin.Context) {
	// 获取URL参数中的文件ID
	fileID := c.Param("id")
	if fileID == "" {
		utils.Warnf("download: file id is required")
		response.ValidationError(c, "file id is required")
		return
	}

	// 调用服务层获取文件内容
	reader, mimeType, size, err := h.fileService.Download(c.Request.Context(), fileID)
	if err != nil {
		utils.Errorf("download: file not found, fileID=%s, error=%v", fileID, err)
		response.Error(c, response.ErrNotFound, "file not found")
		return
	}
	defer reader.Close()

	// 设置响应头，支持在线预览
	c.Header("Content-Type", mimeType)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Content-Disposition", "inline")

	c.DataFromReader(200, size, mimeType, reader, nil)
}

// Delete 删除单个文件
// DELETE /api/v1/file/:id
// 需要用户认证或提供有效的匿名Token
func (h *FileHandler) Delete(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.Warnf("delete: file id is required")
		response.ValidationError(c, "file id is required")
		return
	}

	// 从 cookie 获取 token
	token, _ := c.Cookie("imgbed_token")
	var anonymousID string
	if token != "" {
		claims, _ := utils.ParseToken(token)
		if claims != nil && claims.IsAnonymous {
			anonymousID = claims.AnonymousID
		}
	}

	// 调用服务层删除文件
	if err := h.fileService.Delete(c.Request.Context(), fileID, anonymousID); err != nil {
		utils.Errorf("delete: delete failed, fileID=%s, error=%v", fileID, err)
		if err.Error() == "authorization required" {
			response.Error(c, response.ErrUnauthorized, "authorization required")
		} else {
			response.Error(c, response.ErrInternal, err.Error())
		}
		return
	}

	utils.Infof("delete: success, fileID=%s", fileID)
	response.Success(c, nil)
}

// DeleteMultiple 批量删除文件
// DELETE /api/v1/files
// 需要用户认证，批量删除多个文件
func (h *FileHandler) DeleteMultiple(c *gin.Context) {
	// 解析请求体，获取文件ID列表
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("delete multiple: file ids are required")
		response.ValidationError(c, "file ids are required")
		return
	}

	// 调用服务层批量删除
	success, failed, err := h.fileService.DeleteMultiple(c.Request.Context(), req.IDs)
	if err != nil {
		utils.Errorf("delete multiple: delete failed, ids=%v, error=%v", req.IDs, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("delete multiple: success, success=%d, failed=%d", len(success), len(failed))
	response.Success(c, gin.H{
		"success": success,
		"failed":  failed,
	})
}

// List 获取文件列表
// GET /api/v1/files
// 支持分页、搜索、时间范围筛选和来源筛选，返回文件列表及总数
func (h *FileHandler) List(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	search := c.Query("search")
	source := c.Query("source")

	// 解析时间范围筛选参数
	var startTime, endTime, olderThan int64
	if st := c.Query("startTime"); st != "" {
		startTime, _ = strconv.ParseInt(st, 10, 64)
	}
	if et := c.Query("endTime"); et != "" {
		endTime, _ = strconv.ParseInt(et, 10, 64)
	}
	if ot := c.Query("olderThan"); ot != "" {
		olderThan, _ = strconv.ParseInt(ot, 10, 64)
	}

	// 修正分页参数，确保合法性
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 调用服务层获取文件列表
	files, total, err := h.fileService.List(c.Request.Context(), page, pageSize, search, source, startTime, endTime, olderThan)
	if err != nil {
		utils.Errorf("list: query failed, page=%d, pageSize=%d, error=%v", page, pageSize, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 返回分页数据
	response.Success(c, gin.H{
		"list":     files,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *FileHandler) ListIds(c *gin.Context) {
	search := c.Query("search")
	source := c.Query("source")

	var startTime, endTime, olderThan int64
	if st := c.Query("startTime"); st != "" {
		startTime, _ = strconv.ParseInt(st, 10, 64)
	}
	if et := c.Query("endTime"); et != "" {
		endTime, _ = strconv.ParseInt(et, 10, 64)
	}
	if ot := c.Query("olderThan"); ot != "" {
		olderThan, _ = strconv.ParseInt(ot, 10, 64)
	}

	ids, total, err := h.fileService.ListIds(c.Request.Context(), search, source, startTime, endTime, olderThan)
	if err != nil {
		utils.Errorf("list ids: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	response.Success(c, gin.H{
		"ids":   ids,
		"total": total,
	})
}

// CleanupPreview 预览清理旧文件
// POST /api/v1/files/cleanup/preview
func (h *FileHandler) CleanupPreview(c *gin.Context) {
	var req struct {
		OlderThan int64  `json:"olderThan"` // N天前的文件
		StartTime int64  `json:"startTime"` // 开始时间
		EndTime   int64  `json:"endTime"`   // 结束时间
		ChannelID string `json:"channelID"` // 渠道ID
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("cleanup preview: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	result, err := h.fileService.CleanupPreview(c.Request.Context(), req.OlderThan, req.StartTime, req.EndTime, req.ChannelID)
	if err != nil {
		utils.Errorf("cleanup preview: preview failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("cleanup preview: success, count=%d", result.Count)
	response.Success(c, result)
}

// Cleanup 执行清理旧文件
// POST /api/v1/files/cleanup
func (h *FileHandler) Cleanup(c *gin.Context) {
	var req struct {
		OlderThan int64  `json:"olderThan"` // N天前的文件
		StartTime int64  `json:"startTime"` // 开始时间
		EndTime   int64  `json:"endTime"`   // 结束时间
		ChannelID string `json:"channelID"` // 渠道ID
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("cleanup: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	result, err := h.fileService.Cleanup(c.Request.Context(), req.OlderThan, req.StartTime, req.EndTime, req.ChannelID)
	if err != nil {
		utils.Errorf("cleanup: cleanup failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("cleanup: success, deletedCount=%d, failedCount=%d", result.DeletedCount, result.FailedCount)
	response.Success(c, result)
}

// GetInfo 获取文件详细信息
// GET /api/v1/file/:id/info
// 返回文件的元数据信息，包括大小、类型、创建时间等
// GetInfo 获取文件详情
// @Summary 获取文件详情
// @Description 根据文件ID获取文件的详细信息
// @Tags 文件
// @Produce json
// @Param id path string true "文件ID"
// @Success 200 {object} response.Response{data=model.FileInfo} "获取成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "文件不存在"
// @Router /file/{id}/info [get]
func (h *FileHandler) GetInfo(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.Warnf("get info: file id is required")
		response.ValidationError(c, "file id is required")
		return
	}

	// 调用服务层获取文件详情
	info, err := h.fileService.GetInfo(c.Request.Context(), fileID)
	if err != nil {
		utils.Errorf("get info: file not found, fileID=%s, error=%v", fileID, err)
		response.Error(c, response.ErrNotFound, "file not found")
		return
	}

	response.Success(c, info)
}

// Proxy 代理访问文件
// GET /api/v1/file/:id/proxy
// 通过ID代理访问文件内容，用于CDN加速等场景
func (h *FileHandler) Proxy(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.Warnf("proxy: file id is required")
		response.ValidationError(c, "file id is required")
		return
	}

	// 获取文件内容
	reader, mimeType, size, err := h.fileService.Download(c.Request.Context(), fileID)
	if err != nil {
		utils.Errorf("proxy: file not found, fileID=%s, error=%v", fileID, err)
		response.Error(c, response.ErrNotFound, "file not found")
		return
	}
	defer reader.Close()

	// 设置缓存头，提高访问效率
	c.Header("Content-Type", mimeType)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Cache-Control", "public, max-age=31536000")

	c.DataFromReader(200, size, mimeType, reader, nil)
}

// GetFile 获取文件
// GET /api/v1/file/:id
// 支持扩展名处理，返回文件内容
// @Summary 获取文件
// @Description 根据文件ID获取文件内容
// @Tags 文件
// @Produce octet-stream
// @Param id path string true "文件ID"
// @Success 200 {file} file "文件内容"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "文件不存在"
// @Router /file/{id} [get]
func (h *FileHandler) GetFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.Warnf("get file: file id is required")
		response.ValidationError(c, "file id is required")
		return
	}

	// 获取文件内容
	reader, mimeType, size, err := h.fileService.Download(c.Request.Context(), fileID)
	if err != nil {
		utils.Errorf("get file: file not found, fileID=%s, error=%v", fileID, err)
		response.Error(c, response.ErrNotFound, "file not found")
		return
	}
	defer reader.Close()

	// 处理扩展名，用于设置 Content-Disposition header
	ext := ""
	baseName := fileID
	if idx := strings.LastIndex(fileID, "."); idx != -1 {
		ext = fileID[idx:]
		baseName = fileID[:idx]
	}

	// 设置响应头
	if ext != "" && strings.HasPrefix(mimeType, "image/") {
		c.Header("Content-Disposition", "inline; filename="+baseName+ext)
	} else {
		c.Header("Content-Disposition", "inline")
	}

	c.Header("Content-Type", mimeType)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Cache-Control", "public, max-age=31536000")

	c.DataFromReader(200, size, mimeType, reader, nil)
}

// CheckChecksum 检查文件是否已存在（用于秒传）
// GET /api/v1/file/check/:checksum
func (h *FileHandler) CheckChecksum(c *gin.Context) {
	checksum := c.Param("checksum")
	if checksum == "" {
		response.ValidationError(c, "checksum is required")
		return
	}

	file, err := h.fileService.CheckFileByChecksum(checksum)
	if err != nil {
		utils.Errorf("check checksum: query failed, checksum=%s, error=%v", checksum, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	if file == nil {
		response.Success(c, gin.H{"exists": false})
		return
	}

	response.Success(c, gin.H{
		"exists": true,
		"file": gin.H{
			"id":   file.ID,
			"name": file.Name,
			"url":  file.URL,
			"size": file.Size,
			"type": file.Type,
		},
	})
}
