package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/utils"
)

// TokenHandler API Token公开接口处理器
// 处理公开的Token管理API请求，包括Token创建、列表、删除、启用/禁用等功能
type TokenHandler struct {
	tokenService *service.TokenService // Token服务引用
}

// NewTokenHandler 创建TokenHandler实例
// 返回初始化好的TokenHandler指针
func NewTokenHandler() *TokenHandler {
	return &TokenHandler{
		tokenService: service.NewTokenService(),
	}
}

// ListTokens 获取Token列表
// GET /api/v1/tokens
// 需要用户认证，返回当前用户的所有Token
func (h *TokenHandler) ListTokens(c *gin.Context) {
	tokens, err := h.tokenService.ListTokens(c.Request.Context())
	if err != nil {
		utils.Errorf("list tokens: query failed, error=%v", err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	// 隐藏Secret，只返回Token信息
	result := make([]gin.H, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, gin.H{
			"name":        t.Name,
			"token":       t.Token,
			"permissions": t.Permissions,
			"enabled":     t.Enabled,
			"expiresAt":   t.ExpiresAt.Unix(),
			"createdAt":   t.CreatedAt.Unix(),
			"lastUsedAt":  t.LastUsedAt.Unix(),
		})
	}

	response.Success(c, result)
}

// CreateToken 创建新Token
// POST /api/v1/tokens
// 公开接口，无需认证即可创建（但有限制）
func (h *TokenHandler) CreateToken(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`        // Token名称
		Permissions []string `json:"permissions" binding:"required"` // 权限列表
		ExpiresIn   int      `json:"expiresIn"`                      // 过期天数，0表示永不过期
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("create token: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request: name and permissions are required")
		return
	}

	// 验证权限
	validPermissions := map[string]bool{
		"upload":          true,
		"upload:multiple": true,
		"read":            true,
		"delete":          true,
		"*":               true,
	}
	for _, p := range req.Permissions {
		if !validPermissions[p] {
			utils.Warnf("create token: invalid permission, permission=%s", p)
			response.ValidationError(c, "invalid permission: "+p)
			return
		}
	}

	// 创建Token
	token, err := h.tokenService.CreateToken(c.Request.Context(), req.Name, req.Permissions, req.ExpiresIn)
	if err != nil {
		utils.Errorf("create token: create failed, name=%s, error=%v", req.Name, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("create token: success, name=%s, token=%s", req.Name, token.Token[:8]+"...")

	// 返回完整的Token信息（Secret只显示一次）
	response.Success(c, gin.H{
		"name":        token.Name,
		"token":       token.Token,
		"secret":      token.Secret, // 完整Secret，只在此处返回
		"permissions": token.Permissions,
		"enabled":     token.Enabled,
		"expiresAt":   token.ExpiresAt.Unix(),
		"createdAt":   token.CreatedAt.Unix(),
	})
}

// DeleteToken 删除Token
// DELETE /api/v1/tokens/:token
// 需要用户认证
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	tokenValue := c.Param("token")
	if tokenValue == "" {
		utils.Warnf("delete token: token is required")
		response.ValidationError(c, "token is required")
		return
	}

	// 获取当前用户
	userID, _ := c.Get("userId")

	if err := h.tokenService.DeleteToken(c.Request.Context(), tokenValue); err != nil {
		utils.Errorf("delete token: delete failed, token=%s..., userId=%s, error=%v", tokenValue[:8], userID, err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("delete token: success, token=%s..., userId=%s", tokenValue[:8], userID)
	response.Success(c, nil)
}

// EnableToken 启用或禁用Token
// PUT /api/v1/tokens/:token/enable
// 需要用户认证
func (h *TokenHandler) EnableToken(c *gin.Context) {
	tokenValue := c.Param("token")
	if tokenValue == "" {
		utils.Warnf("enable token: token is required")
		response.ValidationError(c, "token is required")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("enable token: invalid request, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.tokenService.EnableToken(c.Request.Context(), tokenValue, req.Enabled); err != nil {
		utils.Errorf("enable token: update failed, token=%s..., error=%v", tokenValue[:8], err)
		response.Error(c, response.ErrInternal, err.Error())
		return
	}

	utils.Infof("enable token: success, token=%s..., enabled=%v", tokenValue[:8], req.Enabled)
	response.Success(c, gin.H{"token": tokenValue, "enabled": req.Enabled})
}
