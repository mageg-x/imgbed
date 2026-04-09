package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/utils"
)

// BackupService 备份服务
type BackupService struct{}

// NewBackupService 创建备份服务实例
func NewBackupService() *BackupService {
	return &BackupService{}
}

// BackupInfo 备份信息
type BackupInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

// getBackupDir 获取备份目录（相对于数据目录）
func (s *BackupService) getBackupDir() string {
	dbPath := config.GetString("database.path")
	if dbPath == "" {
		dbPath = filepath.Join(config.GetDataDir(), "imgbed.db")
	}
	return filepath.Join(filepath.Dir(dbPath), "backup")
}

// validateBackupPath 校验备份路径安全（防止路径遍历）
func (s *BackupService) validateBackupPath(backupPath string) error {
	backupDir := s.getBackupDir()
	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	absBackupDir, err := filepath.Abs(backupDir)
	if err != nil {
		return fmt.Errorf("invalid backup dir: %w", err)
	}
	if !strings.HasPrefix(absBackupPath, absBackupDir) {
		return fmt.Errorf("invalid backup path: outside backup directory")
	}
	return nil
}

// CreateBackup 创建数据库备份
func (s *BackupService) CreateBackup() (string, error) {
	dbPath := config.GetString("database.path")
	if dbPath == "" {
		dbPath = filepath.Join(config.GetDataDir(), "imgbed.db")
	}

	backupDir := s.getBackupDir()
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("create backup directory failed: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("imgbed-%s.db", timestamp))

	if err := copyFileAtomic(dbPath, backupFile); err != nil {
		return "", fmt.Errorf("copy database file failed: %w", err)
	}

	utils.Infof("backup: created %s", backupFile)
	return backupFile, nil
}

// ListBackups 列出所有备份
func (s *BackupService) ListBackups() ([]BackupInfo, error) {
	backupDir := s.getBackupDir()

	files, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("read backup directory failed: %w", err)
	}

	var backups []BackupInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Name:      file.Name(),
			Path:      file.Name(), // 只返回文件名，不暴露服务器路径
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt > backups[j].CreatedAt
	})

	return backups, nil
}

// DeleteBackup 删除备份
func (s *BackupService) DeleteBackup(backupPath string) error {
	if err := s.validateBackupPath(backupPath); err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("delete backup file failed: %w", err)
	}
	utils.Infof("backup: deleted %s", backupPath)
	return nil
}

// RestoreBackup 从备份恢复
func (s *BackupService) RestoreBackup(backupPath string) error {
	if err := s.validateBackupPath(backupPath); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}

	dbPath := config.GetString("database.path")
	if dbPath == "" {
		dbPath = filepath.Join(config.GetDataDir(), "imgbed.db")
	}

	// 先 checkpoint WAL，确保所有改动写入主数据库
	database.DB.Exec("PRAGMA wal_checkpoint(FULL)")

	if err := copyFileAtomic(backupPath, dbPath); err != nil {
		return fmt.Errorf("restore database file failed: %w", err)
	}

	// 关闭并重新初始化连接，让下次查询读取新数据库
	if err := database.ReinitDB(); err != nil {
		return fmt.Errorf("restore db reinit failed: %w", err)
	}

	utils.Infof("backup: restored from %s", backupPath)
	return nil
}

// AutoBackup 自动备份
func (s *BackupService) AutoBackup() {
	_, err := s.CreateBackup()
	if err != nil {
		utils.Errorf("auto backup failed: %v", err)
	}
}

// copyFileAtomic 原子复制文件：先写临时文件再 rename，避免目标文件被截断
func copyFileAtomic(src, dst string) error {
	tmp := dst + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err = io.Copy(destination, source); err != nil {
		os.Remove(tmp)
		return err
	}
	if err = destination.Sync(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err = destination.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

