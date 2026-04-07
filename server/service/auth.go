package service

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/utils"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务，负责用户和管理员的密码验证、Token生成等功能
type AuthService struct{}

// NewAuthService 创建AuthService实例
func NewAuthService() *AuthService {
	return &AuthService{}
}

// VerifyUserPassword 验证用户密码
// 参数：
//   - password: 用户输入的密码
//
// 返回：
//   - bool: 密码是否正确
//   - error: 验证过程中的错误
func (s *AuthService) VerifyUserPassword(password string) (bool, error) {
	var cfg model.Config
	if err := database.DB.Where("key = ?", "auth.user_password").First(&cfg).Error; err != nil {
		// 数据库查询失败
		utils.Errorf("verify user password: query config failed, key=%s, error=%v", "auth.user_password", err)
		return false, err
	}

	// 如果密码为空，表示未设置密码，验证通过
	if cfg.Value == "" {
		return true, nil
	}

	// 如果是 bcrypt 哈希格式，使用 bcrypt 验证
	if strings.HasPrefix(cfg.Value, "$2") {
		err := bcrypt.CompareHashAndPassword([]byte(cfg.Value), []byte(password))
		if err != nil {
			utils.Warnf("verify user password: password mismatch")
		}
		return err == nil, nil
	}

	// 兼容旧版明文存储的密码（将在下次设置密码时升级为哈希）
	return cfg.Value == password, nil
}

// VerifyAdminPassword 验证管理员用户名和密码
// 参数：
//   - username: 管理员用户名
//   - password: 管理员密码
//
// 返回：
//   - bool: 凭证是否正确
//   - error: 验证过程中的错误
func (s *AuthService) VerifyAdminPassword(username, password string) (bool, error) {
	var usernameCfg, passwordCfg model.Config

	// 查询用户名配置
	if err := database.DB.Where("key = ?", "auth.admin_username").First(&usernameCfg).Error; err != nil {
		utils.Errorf("verify admin password: query username config failed, key=%s, error=%v", "auth.admin_username", err)
		return false, err
	}

	// 验证用户名
	if usernameCfg.Value != username {
		return false, nil
	}

	// 查询密码配置
	if err := database.DB.Where("key = ?", "auth.admin_password").First(&passwordCfg).Error; err != nil {
		utils.Errorf("verify admin password: query password config failed, key=%s, error=%v", "auth.admin_password", err)
		return false, err
	}

	// 如果数据库中没有密码，尝试从环境变量或配置文件获取
	if passwordCfg.Value == "" {
		// 优先使用环境变量
		envPassword := config.GetString("admin.password")
		if envPassword != "" {
			// 使用环境变量中的密码
			if password == envPassword {
				return true, nil
			}
			return false, nil
		}
		// 没有设置密码，拒绝登录
		utils.Warnf("verify admin password: no password configured")
		return false, nil
	}

	// 使用bcrypt验证密码
	err := bcrypt.CompareHashAndPassword([]byte(passwordCfg.Value), []byte(password))
	if err != nil {
		utils.Warnf("verify admin password: password mismatch, username=%s", username)
	}
	return err == nil, nil
}

// SetUserPassword 设置用户密码
// 参数：
//   - password: 要设置的密码
//
// 返回：
//   - error: 设置过程中的错误
func (s *AuthService) SetUserPassword(password string) error {
	return s.setConfig("auth.user_password", password)
}

// SetAdminCredentials 设置管理员凭证（用户名和密码）
// 参数：
//   - username: 管理员用户名
//   - password: 管理员密码（将使用bcrypt加密）
//
// 返回：
//   - error: 设置过程中的错误
func (s *AuthService) SetAdminCredentials(username, password string) error {
	// 使用bcrypt加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		utils.Errorf("set admin credentials: generate hash failed, error=%v", err)
		return err
	}

	// 保存用户名
	if err := s.setConfig("auth.admin_username", username); err != nil {
		utils.Errorf("set admin credentials: save username failed, key=%s, error=%v", "auth.admin_username", err)
		return err
	}

	// 保存加密后的密码
	return s.setConfig("auth.admin_password", string(hashedPassword))
}

// GenerateUserToken 生成用户Token
// 返回：
//   - string: 生成的JWT token
//   - error: 生成过程中的错误
func (s *AuthService) GenerateUserToken() (string, error) {
	return utils.GenerateToken("user", "user", "user")
}

// GenerateAdminToken 生成管理员Token
// 参数：
//   - username: 管理员用户名
//
// 返回：
//   - string: 生成的JWT token
//   - error: 生成过程中的错误
func (s *AuthService) GenerateAdminToken(username string) (string, error) {
	return utils.GenerateToken("admin", username, "admin")
}

// setConfig 内部方法：设置配置项的值（如果不存在则创建，存在则更新）
// 参数：
//   - key: 配置项的键
//   - value: 配置项的值
//
// 返回：
//   - error: 设置过程中的错误
func (s *AuthService) setConfig(key, value string) error {
	var cfg model.Config
	result := database.DB.Where("key = ?", key).First(&cfg)
	if result.Error != nil {
		// 配置项不存在，创建新的
		cfg.Key = key
		cfg.Value = value
		if err := database.DB.Create(&cfg).Error; err != nil {
			utils.Errorf("set config: create config failed, key=%s, error=%v", key, err)
			return err
		}
		return nil
	}

	// 配置项已存在，更新值
	cfg.Value = value
	if err := database.DB.Save(&cfg).Error; err != nil {
		utils.Errorf("set config: update config failed, key=%s, error=%v", key, err)
		return err
	}
	return nil
}

// GetSessionTimeout 获取会话超时时间（秒）
// 返回：
//   - int: 会话超时时间，默认为86400秒（24小时）
func (s *AuthService) GetSessionTimeout() int {
	var cfg model.Config
	if err := database.DB.Where("key = ?", "auth.session_timeout").First(&cfg).Error; err != nil {
		// 查询失败，返回默认值
		utils.Warnf("get session timeout: query config failed, use default 86400, error=%v", err)
		return 86400
	}

	// 解析超时时间值
	timeout := 0
	for _, c := range cfg.Value {
		if c >= '0' && c <= '9' {
			timeout = timeout*10 + int(c-'0')
		}
	}

	// 如果解析结果为0，使用默认值
	if timeout == 0 {
		return 86400
	}
	return timeout
}

// IsAdminInitialized 检查管理员是否已初始化
// 返回：
//   - bool: 管理员是否已设置密码
func (s *AuthService) IsAdminInitialized() bool {
	var passwordCfg model.Config
	if err := database.DB.Where("key = ?", "auth.admin_password").First(&passwordCfg).Error; err != nil {
		return false
	}
	return passwordCfg.Value != ""
}

// ValidatePasswordStrength 验证密码强度
// 参数：
//   - password: 待验证的密码
//
// 返回：
//   - bool: 密码是否符合强度要求
//   - string: 不符合要求时的提示信息
func (s *AuthService) ValidatePasswordStrength(password string) (bool, string) {
	if len(password) < 8 {
		return false, "密码长度至少为8位"
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper {
		return false, "密码必须包含至少一个大写字母"
	}
	if !hasLower {
		return false, "密码必须包含至少一个小写字母"
	}
	if !hasDigit {
		return false, "密码必须包含至少一个数字"
	}

	return true, ""
}

// InitAdmin 初始化管理员账户
// 参数：
//   - username: 管理员用户名
//   - password: 管理员密码
//
// 返回：
//   - error: 初始化过程中的错误
func (s *AuthService) InitAdmin(username, password string) error {
	// 验证密码强度
	if valid, msg := s.ValidatePasswordStrength(password); !valid {
		return fmt.Errorf("密码强度不足: %s", msg)
	}

	return s.SetAdminCredentials(username, password)
}
