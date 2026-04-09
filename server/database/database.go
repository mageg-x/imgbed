package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB 全局数据库连接实例
// 使用GORM ORM框架，支持SQLite数据库
var DB *gorm.DB

// Init 初始化数据库连接
// 创建数据库文件目录、建立连接、执行自动迁移、初始化默认数据
// 返回：
//   - error: 初始化失败时的错误
func Init() error {
	var err error

	// 获取数据库文件路径
	dbPath := config.GetString("database.path")

	// 确保数据库文件目录存在
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		utils.Errorf("database init: create directory failed, path=%s, error=%v", dbDir, err)
		return fmt.Errorf("create database directory failed: %w", err)
	}

	// 配置GORM日志级别
	gormConfig := &gorm.Config{}
	if config.GetString("app.mode") == "debug" {
		// 调试模式：显示SQL语句
		gormConfig.Logger = gormlogger.Default.LogMode(gormlogger.Info)
	} else {
		// 生产模式：静默日志
		gormConfig.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	// 打开数据库连接
	DB, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		utils.Errorf("database init: connect failed, path=%s, error=%v", dbPath, err)
		return fmt.Errorf("connect database failed: %w", err)
	}

	// 执行自动迁移
	if err = autoMigrate(); err != nil {
		utils.Errorf("database init: migrate failed, error=%v", err)
		return fmt.Errorf("migrate database failed: %w", err)
	}

	// 初始化默认数据
	if err = initDefaultData(); err != nil {
		utils.Errorf("database init: init default data failed, error=%v", err)
		return fmt.Errorf("init default data failed: %w", err)
	}

	// 初始化 FTS5 全文搜索（可选，失败时降级到 LIKE）
	initFTS5()

	utils.Info("database initialized successfully")
	return nil
}

// autoMigrate 执行数据库自动迁移
// 自动创建或更新数据表结构，保持与模型定义一致
// 返回：
//   - error: 迁移失败时的错误
func autoMigrate() error {
	return DB.AutoMigrate(
		&model.File{},       // 文件表
		&model.Channel{},    // 存储渠道表
		&model.Tag{},        // 标签表
		&model.APIToken{},   // API Token表
		&model.Config{},     // 系统配置表
		&model.FileAccess{}, // 文件访问记录表
	)
}

// initFTS5 初始化 FTS5 全文搜索
// 创建 files_fts 虚拟表和触发器，用于高效模糊搜索
// 如果 FTS5 不可用，优雅降级到 LIKE 搜索
func initFTS5() error {
	// 尝试创建 FTS5 虚拟表（存储 file ID 而不是 rowid，因为 File.ID 是 string 类型）
	createFTS := `
	CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
		id,
		name,
		original_name
	);
	`
	if err := DB.Exec(createFTS).Error; err != nil {
		utils.Warnf("FTS5 not available, will use LIKE search: %v", err)
		return nil // 优雅降级，不阻止启动
	}

	// 插入已有数据到 FTS 表
	insertFTS := `
	INSERT OR REPLACE INTO files_fts(id, name, original_name)
	SELECT id, name, original_name FROM files;
	`
	DB.Exec(insertFTS)

	// 创建触发器：插入时同步
	createInsertTrigger := `
	CREATE TRIGGER IF NOT EXISTS files_ai AFTER INSERT ON files BEGIN
		INSERT INTO files_fts(id, name, original_name) VALUES (NEW.id, NEW.name, NEW.original_name);
	END;
	`
	DB.Exec(createInsertTrigger)

	// 创建触发器：删除时同步
	createDeleteTrigger := `
	CREATE TRIGGER IF NOT EXISTS files_ad AFTER DELETE ON files BEGIN
		DELETE FROM files_fts WHERE id = OLD.id;
	END;
	`
	DB.Exec(createDeleteTrigger)

	// 创建触发器：更新时同步
	createUpdateTrigger := `
	CREATE TRIGGER IF NOT EXISTS files_au AFTER UPDATE ON files BEGIN
		UPDATE files_fts SET name = NEW.name, original_name = NEW.original_name WHERE id = OLD.id;
	END;
	`
	DB.Exec(createUpdateTrigger)

	utils.Info("FTS5 search initialized")
	return nil
}

// initDefaultData 初始化默认配置数据
// 确保所有配置项都存在于数据库中，不存在则创建
// 返回：
//   - error: 初始化失败时的错误
func initDefaultData() error {
	// 默认配置项列表
	defaultConfigs := []struct {
		Key   string
		Value string
	}{
		// 应用配置
		{"app.host", "0.0.0.0"},
		{"app.port", "8380"},
		{"app.mode", "debug"},

		// JWT配置
		{"jwt.secret", "imgbed-secret-key"},
		{"jwt.expire", "86400"},

		// 认证配置
		{"auth.user_password", ""},
		{"auth.admin_username", "admin"},
		{"auth.admin_password", "$2a$10$GgdSbSG8qlo8ez.7ROiVJOI5Ex958rcX/3zM80wuGjujUKmWXamxm"}, // 密码: admin
		{"auth.session_timeout", "86400"},

		// 上传配置
		{"upload.max_size", "20971520"},
		{"upload.chunk_size", "5242880"},
		{"upload.default_channel", ""},
		{"upload.allowed_types", "image/*"},
		{"upload.auto_retry", "true"},
		{"upload.retry_count", "3"},

		// 图片压缩配置
		{"compression.enabled", "true"},
		{"compression.quality", "80"},
		{"compression.format", "webp"},
		{"compression.max_width", "1920"},
		{"compression.max_height", "1080"},

		// 匿名上传限制
		{"anonymous.enabled", "true"},
		{"anonymous.rate_limit", "10"},
		{"anonymous.daily_limit", "100"},
		{"anonymous.max_file_size", "5242880"},

		// 调度策略
		{"schedule.strategy", "priority"},

		// 速率限制
		{"rate_limit.enabled", "true"},
		{"rate_limit.rate_limit", "10"},
		{"rate_limit.daily_limit", "100"},
		{"rate_limit.max_file_size", "5242880"},

		// 内容审核
		{"moderation.enabled", "false"},
		{"moderation.provider", ""},
		{"moderation.api_key", ""},

		// 站点配置
		{"site.name", "ImgBed"},
		{"site.logo", ""},
	}

	// 确保每个配置项都存在
	for _, cfg := range defaultConfigs {
		var existing model.Config
		result := DB.Where("key = ?", cfg.Key).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			// 不存在，创建
			if err := DB.Create(&model.Config{Key: cfg.Key, Value: cfg.Value}).Error; err != nil {
				utils.Errorf("init default data: create config failed, key=%s, error=%v", cfg.Key, err)
				return err
			}
			utils.Debugf("init default data: created config %s=%s", cfg.Key, cfg.Value)
		}
	}

	utils.Info("default config data initialized")
	return nil
}

// GetDB 获取数据库连接实例
// 返回：
//   - *gorm.DB: 数据库连接实例
func GetDB() *gorm.DB {
	return DB
}

// CloseDB 关闭数据库连接（用于备份恢复后重置连接）
func CloseDB() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// ReinitDB 重新初始化数据库连接（备份恢复后调用）
func ReinitDB() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
	return Init()
}
