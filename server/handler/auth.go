package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/utils"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: service.NewAuthService(),
	}
}

type LoginRequest struct {
	Password string `json:"password"`
}

type AdminLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("login: parse request failed, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	valid, err := h.authService.VerifyUserPassword(req.Password)
	if err != nil {
		utils.Errorf("login: verify password failed, error=%v", err)
		response.Error(c, response.ErrInternal, "verify password failed")
		return
	}

	if !valid {
		utils.Warnf("login: invalid password")
		response.Error(c, response.ErrUnauthorized, "invalid password")
		return
	}

	token, err := h.authService.GenerateUserToken()
	if err != nil {
		utils.Errorf("login: generate token failed, error=%v", err)
		response.Error(c, response.ErrInternal, "generate token failed")
		return
	}

	c.SetCookie("imgbed_token", token, h.authService.GetSessionTimeout(), "/", "", false, true)
	utils.Infof("login: success")
	response.Success(c, gin.H{"token": token})
}

func (h *AuthHandler) AdminLogin(c *gin.Context) {
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("admin login: parse request failed, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	valid, err := h.authService.VerifyAdminPassword(req.Username, req.Password)
	if err != nil {
		utils.Errorf("admin login: verify password failed, username=%s, error=%v", req.Username, err)
		response.Error(c, response.ErrInternal, "verify password failed")
		return
	}

	if !valid {
		utils.Warnf("admin login: invalid credentials, username=%s", req.Username)
		response.Error(c, response.ErrUnauthorized, "invalid credentials")
		return
	}

	token, err := h.authService.GenerateAdminToken(req.Username)
	if err != nil {
		utils.Errorf("admin login: generate token failed, username=%s, error=%v", req.Username, err)
		response.Error(c, response.ErrInternal, "generate token failed")
		return
	}

	c.SetCookie("imgbed_token", token, h.authService.GetSessionTimeout(), "/", "", false, true)
	utils.Infof("admin login: success, username=%s", req.Username)
	response.Success(c, gin.H{"token": token})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("imgbed_token", "", -1, "/", "", false, true)
	utils.Infof("logout: success")
	response.Success(c, nil)
}

func (h *AuthHandler) SetPassword(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("set password: parse request failed, error=%v", err)
		response.ValidationError(c, "invalid password")
		return
	}

	role, _ := c.Get("role")
	if role != "admin" {
		utils.Warnf("set password: admin access required, role=%v", role)
		response.Error(c, response.ErrForbidden, "admin access required")
		return
	}

	if err := h.authService.SetUserPassword(req.Password); err != nil {
		utils.Errorf("set password: service failed, error=%v", err)
		response.Error(c, response.ErrInternal, "set password failed")
		return
	}

	utils.Infof("set password: success")
	response.Success(c, nil)
}

func (h *AuthHandler) SetAdminPassword(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warnf("set admin password: parse request failed, error=%v", err)
		response.ValidationError(c, "invalid request")
		return
	}

	if err := h.authService.SetAdminCredentials(req.Username, req.Password); err != nil {
		utils.Errorf("set admin password: service failed, username=%s, error=%v", req.Username, err)
		response.Error(c, response.ErrInternal, "set admin credentials failed")
		return
	}

	utils.Infof("set admin password: success, username=%s", req.Username)
	response.Success(c, nil)
}

func (h *AuthHandler) VerifyToken(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		utils.Warnf("verify token: token required")
		response.Error(c, response.ErrUnauthorized, "token required")
		return
	}

	token = strings.TrimPrefix(token, "Bearer ")

	userId, _ := c.Get("userId")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	response.Success(c, gin.H{
		"valid":    true,
		"userId":   userId,
		"username": username,
		"role":     role,
	})
}

func (h *AuthHandler) CheckAuth(c *gin.Context) {
	userId, _ := c.Get("userId")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	response.Success(c, gin.H{
		"authenticated": true,
		"userId":        userId,
		"username":      username,
		"role":          role,
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	userId, _ := c.Get("userId")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	token, err := h.authService.GenerateUserToken()
	if err != nil {
		utils.Errorf("refresh token: generate token failed, error=%v", err)
		response.Error(c, response.ErrInternal, "generate token failed")
		return
	}

	c.SetCookie("imgbed_token", token, h.authService.GetSessionTimeout(), "/", "", false, true)
	response.Success(c, gin.H{
		"token":    token,
		"userId":   userId,
		"username": username,
		"role":     role,
	})
}

func (h *AuthHandler) Status(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "ok",
	})
}

func (h *AuthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}
