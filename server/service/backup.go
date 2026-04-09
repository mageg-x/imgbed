package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/imgbed/server/config"
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

// CreateBackup 创建数据库备份
func (s *BackupService) CreateBackup() (string, error) {
	dbPath := config.GetString("database.path")
	if dbPath == "" {
		dbPath = filepath.Join(config.GetDataDir(), "imgbed.db")
	}

	// 确保备份目录存在
	backupDir := filepath.Join(filepath.Dir(dbPath), "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("create backup directory failed: %w", err)
	}

	// 生成备份文件名
	timestamp := time.Now().Format("20060102150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("imgbed-%s.db", timestamp))

	// 复制数据库文件
	if err := copyFile(dbPath, backupFile); err != nil {
		return "", fmt.Errorf("copy database file failed: %w", err)
	}

	return backupFile, nil
}

// ListBackups 列出所有备份
func (s *BackupService) ListBackups() ([]BackupInfo, error) {
	dbPath := config.GetString("database.path")
	if dbPath == "" {
		dbPath = filepath.Join(config.GetDataDir(), "imgbed.db")
	}

	backupDir := filepath.Join(filepath.Dir(dbPath), "backup")

	// 读取备份目录
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

		filePath := filepath.Join(backupDir, file.Name())
		info, err := file.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Name:      file.Name(),
			Path:      filePath,
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// 按创建时间倒序排序
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt > backups[j].CreatedAt
	})

	return backups, nil
}

// DeleteBackup 删除备份
func (s *BackupService) DeleteBackup(backupPath string) error {
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("delete backup file failed: %w", err)
	}
	return nil
}

// RestoreBackup 从备份恢复
func (s *BackupService) RestoreBackup(backupPath string) error {
	dbPath := config.GetString("database.path")
	if dbPath == "" {
		dbPath = filepath.Join(config.GetDataDir(), "imgbed.db")
	}

	// 复制备份文件到数据库位置
	if err := copyFile(backupPath, dbPath); err != nil {
		return fmt.Errorf("restore database file failed: %w", err)
	}

	return nil
}

// AutoBackup 自动备份
func (s *BackupService) AutoBackup() {
	_, err := s.CreateBackup()
	if err != nil {
		fmt.Printf("auto backup failed: %v\n", err)
	}
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
