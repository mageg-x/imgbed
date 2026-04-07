package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/utils"
)

type TokenService struct{}

func NewTokenService() *TokenService {
	return &TokenService{}
}

type APIToken = model.APIToken

var secretKey = []byte("imgbed-token-secret-key-32byte")

func encryptSecret(plaintext string) (string, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func decryptSecret(ciphertext string) (string, error) {
	data, err := hex.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", err
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (s *TokenService) CreateToken(ctx context.Context, name string, permissions []string, expiresIn int) (*model.APIToken, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		utils.Errorf("create token: generate token bytes failed, name=%s, error=%v", name, err)
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)

	secretBytes := make([]byte, 16)
	if _, err := rand.Read(secretBytes); err != nil {
		utils.Errorf("create token: generate secret bytes failed, name=%s, error=%v", name, err)
		return nil, err
	}
	secret := hex.EncodeToString(secretBytes)

	encryptedSecret, err := encryptSecret(secret)
	if err != nil {
		utils.Errorf("create token: encrypt secret failed, name=%s, error=%v", name, err)
		return nil, err
	}

	expiresAt := time.Time{}
	if expiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(expiresIn) * 24 * time.Hour)
	}

	apiToken := &model.APIToken{
		Token:       token,
		Secret:      encryptedSecret,
		Name:        name,
		Permissions: strings.Join(permissions, ","),
		ExpiresAt:   expiresAt,
		Enabled:     true,
	}

	if err := database.DB.Create(apiToken).Error; err != nil {
		utils.Errorf("create token: save to database failed, name=%s, error=%v", name, err)
		return nil, err
	}

	apiToken.Secret = secret
	return apiToken, nil
}

func (s *TokenService) GetToken(ctx context.Context, token string) (*model.APIToken, error) {
	var apiToken model.APIToken
	if err := database.DB.Where("token = ?", token).First(&apiToken).Error; err != nil {
		utils.Errorf("get token: query failed, token=%s, error=%v", token[:minLen(len(token), 8)]+"...", err)
		return nil, err
	}
	return &apiToken, nil
}

func (s *TokenService) ListTokens(ctx context.Context) ([]model.APIToken, error) {
	var tokens []model.APIToken
	if err := database.DB.Order("created_at DESC").Find(&tokens).Error; err != nil {
		utils.Errorf("list tokens: query failed, error=%v", err)
		return nil, err
	}
	return tokens, nil
}

func (s *TokenService) DeleteToken(ctx context.Context, token string) error {
	if err := database.DB.Where("token = ?", token).Delete(&model.APIToken{}).Error; err != nil {
		utils.Errorf("delete token: delete failed, token=%s, error=%v", token[:minLen(len(token), 8)]+"...", err)
		return err
	}
	return nil
}

func (s *TokenService) EnableToken(ctx context.Context, token string, enabled bool) error {
	if err := database.DB.Model(&model.APIToken{}).Where("token = ?", token).Update("enabled", enabled).Error; err != nil {
		utils.Errorf("enable token: update failed, token=%s, enabled=%v, error=%v", token[:minLen(len(token), 8)]+"...", enabled, err)
		return err
	}
	return nil
}

func (s *TokenService) ValidateToken(ctx context.Context, token, secret string) (*model.APIToken, error) {
	var apiToken model.APIToken
	if err := database.DB.Where("token = ?", token).First(&apiToken).Error; err != nil {
		utils.Errorf("validate token: query failed, token=%s, error=%v", token[:minLen(len(token), 8)]+"...", err)
		return nil, ErrTokenInvalid
	}

	decryptedSecret, err := decryptSecret(apiToken.Secret)
	if err != nil {
		utils.Errorf("validate token: decrypt secret failed, token=%s, error=%v", token[:minLen(len(token), 8)]+"...", err)
		return nil, ErrTokenInvalid
	}

	if decryptedSecret != secret {
		utils.Warnf("validate token: secret mismatch, token=%s", token[:minLen(len(token), 8)]+"...")
		return nil, ErrTokenInvalid
	}

	if !apiToken.Enabled {
		utils.Warnf("validate token: token is disabled, token=%s", token[:minLen(len(token), 8)]+"...")
		return nil, ErrTokenDisabled
	}

	if !apiToken.ExpiresAt.IsZero() && time.Now().After(apiToken.ExpiresAt) {
		utils.Warnf("validate token: token expired, token=%s, expiresAt=%v", token[:minLen(len(token), 8)]+"...", apiToken.ExpiresAt)
		return nil, ErrTokenExpired
	}

	if err := database.DB.Model(&apiToken).Update("last_used_at", time.Now()).Error; err != nil {
		utils.Warnf("validate token: update last_used_at failed, token=%s, error=%v", token[:minLen(len(token), 8)]+"...", err)
	}

	return &apiToken, nil
}

func (s *TokenService) HasPermission(token *model.APIToken, permission string) bool {
	permissions := strings.Split(token.Permissions, ",")
	for _, p := range permissions {
		p = strings.TrimSpace(p)
		if p == "*" || p == permission {
			return true
		}
	}
	return false
}

var (
	ErrTokenDisabled = &TokenError{Code: response.ErrCodeTokenDisabled, Message: "token is disabled"}
	ErrTokenExpired  = &TokenError{Code: response.ErrCodeTokenExpired, Message: "token has expired"}
	ErrTokenInvalid  = &TokenError{Code: response.ErrCodeTokenInvalid, Message: "invalid token"}
)

type TokenError struct {
	Code    int
	Message string
}

func (e *TokenError) Error() string {
	return e.Message
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
