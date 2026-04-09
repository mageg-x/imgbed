package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/service"
)

// BackupHandler 备份处理器
type BackupHandler struct {
	backupService *service.BackupService
}

// NewBackupHandler 创建备份处理器实例
func NewBackupHandler(backupService *service.BackupService) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
	}
}

// CreateBackup 创建备份
func (h *BackupHandler) CreateBackup(c *gin.Context) {
	backupFile, err := h.backupService.CreateBackup()
	if err != nil {
		response.Error(c, response.ErrInternal, "create backup failed")
		return
	}

	response.Success(c, gin.H{
		"backup_file": backupFile,
	})
}

// ListBackups 列出备份
func (h *BackupHandler) ListBackups(c *gin.Context) {
	backups, err := h.backupService.ListBackups()
	if err != nil {
		response.Error(c, response.ErrInternal, "list backups failed")
		return
	}

	response.Success(c, backups)
}

// DeleteBackup 删除备份
func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	var req struct {
		BackupPath string `json:"backup_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.backupService.DeleteBackup(req.BackupPath); err != nil {
		response.Error(c, response.ErrInternal, "delete backup failed")
		return
	}

	response.Success(c, nil)
}

// RestoreBackup 恢复备份
func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	var req struct {
		BackupPath string `json:"backup_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.backupService.RestoreBackup(req.BackupPath); err != nil {
		response.Error(c, response.ErrInternal, "restore backup failed")
		return
	}

	response.Success(c, nil)
}
