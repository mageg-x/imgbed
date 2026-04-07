package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT相关错误定义
var (
	// ErrTokenExpired Token已过期
	ErrTokenExpired = errors.New("token has expired")
	// ErrTokenInvalid Token无效
	ErrTokenInvalid = errors.New("token is invalid")
	// ErrTokenMalformed Token格式错误
	ErrTokenMalformed = errors.New("token is malformed")
	// ErrTokenNotValidYet Token尚未生效
	ErrTokenNotValidYet = errors.New("token is not valid yet")

	// JWT配置（由SetJWTConfig设置）
	jwtSecret string // JWT签名密钥
	jwtExpire int    // Token过期时间（秒）
)

// SetJWTConfig 设置JWT配置
// 在应用启动时由main.go调用，配置JWT签名密钥和过期时间
// 参数：
//   - secret: JWT签名密钥
//   - expire: Token过期时间（秒）
func SetJWTConfig(secret string, expire int) {
	jwtSecret = secret
	jwtExpire = expire
}

// UpdateJWTConfig 更新JWT配置（运行时）
// 参数：
//   - secret: 新JWT签名密钥（空字符串则不更新）
//   - expire: 新Token过期时间（0则不更新）
func UpdateJWTConfig(secret string, expire int) {
	if secret != "" {
		jwtSecret = secret
	}
	if expire > 0 {
		jwtExpire = expire
	}
}

// GetJWTConfig 获取当前JWT配置
func GetJWTConfig() (secret string, expire int) {
	return jwtSecret, jwtExpire
}

// JWTClaims JWT声明结构
// 包含用户信息和标准JWT声明
type JWTClaims struct {
	UserID       string `json:"userId"`   // 用户ID
	Username     string `json:"username"` // 用户名
	Role         string `json:"role"`     // 用户角色
	IsAnonymous  bool   `json:"isAnonymous"`  // 是否匿名用户
	AnonymousID  string `json:"anonymousId"` // 匿名用户ID
	jwt.RegisteredClaims          // 标准JWT声明
}

// GenerateToken 生成JWT Token
// 根据用户信息生成包含用户ID、用户名、角色的JWT Token
// 参数：
//   - userID: 用户ID
//   - username: 用户名
//   - role: 用户角色
//
// 返回：
//   - string: JWT Token字符串
//   - error: 生成失败时的错误
func GenerateToken(userID, username, role string) (string, error) {
	secret := jwtSecret
	expire := jwtExpire
	// 默认过期时间为24小时
	if expire == 0 {
		expire = 86400
	}

	// 构建JWT声明
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "imgbed",
		},
	}

	// 使用HS256算法签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		logLogger.Errorf("generate token: sign failed, error=%v", err)
		return "", err
	}
	logLogger.Debugf("generate token: success, userID=%s, role=%s", userID, role)
	return tokenString, nil
}

// GenerateAnonymousToken 生成匿名用户JWT Token（30天有效期）
func GenerateAnonymousToken(anonymousID string) (string, error) {
	secret := jwtSecret
	expire := 30 * 24 * 3600 // 30天

	claims := JWTClaims{
		UserID:      anonymousID,
		Username:    "anonymous",
		Role:        "anonymous",
		IsAnonymous: true,
		AnonymousID: anonymousID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "imgbed",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		logLogger.Errorf("generate anonymous token: sign failed, error=%v", err)
		return "", err
	}
	logLogger.Debugf("generate anonymous token: success, anonymousID=%s", anonymousID)
	return tokenString, nil
}

// ParseToken 解析并验证JWT Token
// 验证Token签名、过期时间等，返回解析后的声明
// 参数：
//   - tokenString: JWT Token字符串
//
// 返回：
//   - *JWTClaims: 解析后的JWT声明
//   - error: 解析或验证失败时的错误
func ParseToken(tokenString string) (*JWTClaims, error) {
	secret := jwtSecret
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		// 根据错误类型返回对应的错误
		if errors.Is(err, jwt.ErrTokenExpired) {
			logLogger.Warnf("parse token: token expired")
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			logLogger.Warnf("parse token: token malformed")
			return nil, ErrTokenMalformed
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			logLogger.Warnf("parse token: token not valid yet")
			return nil, ErrTokenNotValidYet
		}
		logLogger.Errorf("parse token: invalid token, error=%v", err)
		return nil, ErrTokenInvalid
	}

	// 验证成功，返回声明
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		logLogger.Debugf("parse token: success, userID=%s, role=%s", claims.UserID, claims.Role)
		return claims, nil
	}
	logLogger.Warnf("parse token: invalid claims")
	return nil, ErrTokenInvalid
}
