package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/imgbed/server/config"
	"github.com/imgbed/server/response"
	"github.com/imgbed/server/service"
	"github.com/imgbed/server/utils"
	"go.uber.org/zap"
)

const (
	MaxFileSize      = 20 * 1024 * 1024
	DefaultRateLimit = 60
	RequestIDHeader  = "X-Request-ID"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("requestId", requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		requestID, _ := c.Get("requestId")

		utils.Info("request",
			zap.String("requestId", requestID.(string)),
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user-agent", c.Request.UserAgent()),
		)
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				utils.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
				)
				response.Error(c, response.ErrInternal, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		allowedOrigins := config.GetStringSlice("cors.allowedOrigins")

		if len(allowedOrigins) > 0 {
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
				if strings.HasPrefix(allowedOrigin, "*.") {
					domain := allowedOrigin[2:]
					if strings.HasSuffix(origin, domain) || origin == "http://"+domain || origin == "https://"+domain {
						allowed = true
						break
					}
				}
			}

			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		} else {
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With, X-API-Token, X-API-Secret")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Disposition")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tryJWTAuth(c) || tryTokenAuth(c) {
			c.Next()
			return
		}

		utils.Warnf("auth required: unauthorized request, path=%s, ip=%s", c.Request.URL.Path, c.ClientIP())
		response.Error(c, response.ErrUnauthorized, "authorization required")
		c.Abort()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tryJWTAuth(c) {
			role, _ := c.Get("role")
			if role == "admin" {
				c.Next()
				return
			}
			utils.Warnf("admin required: insufficient permissions, role=%v, path=%s", role, c.Request.URL.Path)
			response.Error(c, response.ErrForbidden, "admin access required")
			c.Abort()
			return
		}

		if tryTokenAuth(c) {
			role, _ := c.Get("role")
			if role == "admin" {
				c.Next()
				return
			}
		}

		utils.Warnf("admin required: unauthorized access, path=%s, ip=%s", c.Request.URL.Path, c.ClientIP())
		response.Error(c, response.ErrForbidden, "admin access required")
		c.Abort()
	}
}

func tryJWTAuth(c *gin.Context) bool {
	token := c.GetHeader("Authorization")
	if token == "" {
		token, _ = c.Cookie("imgbed_token")
	}
	if token == "" {
		return false
	}

	token = strings.TrimPrefix(token, "Bearer ")

	claims, err := utils.ParseToken(token)
	if err != nil {
		utils.Debugf("jwt auth: parse token failed, error=%v", err)
		return false
	}

	c.Set("userId", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("role", claims.Role)
	c.Set("authType", "jwt")
	return true
}

func tryTokenAuth(c *gin.Context) bool {
	apiToken := c.GetHeader("X-API-Token")
	apiSecret := c.GetHeader("X-API-Secret")

	if apiToken == "" || apiSecret == "" {
		return false
	}

	tokenService := service.NewTokenService()
	token, err := tokenService.ValidateToken(c.Request.Context(), apiToken, apiSecret)
	if err != nil {
		utils.Warnf("token auth: validation failed, token=%s..., error=%v", apiToken[:min(8, len(apiToken))], err)
		return false
	}

	c.Set("userId", "token:"+token.Token[:min(8, len(token.Token))])
	c.Set("username", token.Name)
	c.Set("role", "user")
	c.Set("authType", "token")
	c.Set("apiToken", token)
	return true
}

func TokenPermissionRequired(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authType, _ := c.Get("authType")
		if authType != "token" {
			c.Next()
			return
		}

		apiToken, exists := c.Get("apiToken")
		if !exists {
			utils.Warnf("token permission: token not found in context, path=%s", c.Request.URL.Path)
			response.Error(c, response.ErrUnauthorized, "invalid token")
			c.Abort()
			return
		}

		token := apiToken.(*service.APIToken)
		tokenService := service.NewTokenService()
		if !tokenService.HasPermission(token, permission) {
			utils.Warnf("token permission: denied, token=%s..., permission=%s, path=%s", token.Token[:min(8, len(token.Token))], permission, c.Request.URL.Path)
			response.Error(c, response.ErrForbidden, "permission denied")
			c.Abort()
			return
		}

		c.Next()
	}
}

type rateLimiter struct {
	sync.RWMutex
	visitors map[string]*visitor
}

type visitor struct {
	lastSeen time.Time
	count    int
}

var limiter = rateLimiter{
	visitors: make(map[string]*visitor),
}

func init() {
	go cleanupVisitors()
}

func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		limiter.Lock()
		for ip, v := range limiter.visitors {
			if time.Since(v.lastSeen) > time.Minute {
				delete(limiter.visitors, ip)
			}
		}
		limiter.Unlock()
	}
}

func RateLimitFromConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.GetBool("rate_limit.enabled") {
			c.Next()
			return
		}

		rateLimit := config.GetInt("rate_limit.rate_limit")
		if rateLimit <= 0 {
			rateLimit = 10
		}

		ip := c.ClientIP()

		limiter.Lock()
		v, exists := limiter.visitors[ip]
		if !exists {
			limiter.visitors[ip] = &visitor{
				lastSeen: time.Now(),
				count:    1,
			}
			limiter.Unlock()
			c.Next()
			return
		}

		if time.Since(v.lastSeen) > time.Minute {
			v.count = 1
			v.lastSeen = time.Now()
		} else {
			v.count++
			v.lastSeen = time.Now()
		}
		limiter.Unlock()

		if v.count > rateLimit {
			utils.Warnf("rate limit: exceeded, ip=%s, count=%d, limit=%d, path=%s", ip, v.count, rateLimit, c.Request.URL.Path)
			response.Error(c, response.ErrTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

func RateLimit(requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		limiter.Lock()
		v, exists := limiter.visitors[ip]
		if !exists {
			limiter.visitors[ip] = &visitor{
				lastSeen: time.Now(),
				count:    1,
			}
			limiter.Unlock()
			c.Next()
			return
		}

		if time.Since(v.lastSeen) > time.Minute {
			v.count = 1
			v.lastSeen = time.Now()
		} else {
			v.count++
			v.lastSeen = time.Now()
		}
		limiter.Unlock()

		if v.count > requestsPerMinute {
			utils.Warnf("rate limit: exceeded, ip=%s, count=%d, limit=%d, path=%s", ip, v.count, requestsPerMinute, c.Request.URL.Path)
			response.Error(c, response.ErrTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// 静态落地页（i-want.html、guide.html）需要宽松的 CSP，允许外部资源
		path := c.Request.URL.Path
		if path == "/i-want.html" || path == "/guide.html" || path == "/" {
			csp := "default-src 'self'; " +
				"img-src 'self' data: blob: https:; " +
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
				"font-src 'self' data: https://fonts.gstatic.com; " +
				"connect-src 'self' https://gh-proxy.ma3ok.com https://gh-proxy.com https://api.github.com; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'"
			c.Header("Content-Security-Policy", csp)
		} else {
			csp := "default-src 'self'; " +
				"img-src 'self' data: blob: https:; " +
				"style-src 'self' 'unsafe-inline'; " +
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
				"font-src 'self' data:; " +
				"connect-src 'self'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'"
			c.Header("Content-Security-Policy", csp)
		}

		c.Header("X-Powered-By", "")
		c.Header("Server", "")

		c.Next()
	}
}

func SecureFileDownload() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}

func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if gin.IsDebugging() {
			c.Next()
			return
		}

		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		referer := c.GetHeader("Referer")
		host := c.Request.Host

		if origin == "" && referer == "" {
			if c.GetHeader("X-API-Token") != "" {
				c.Next()
				return
			}
			utils.Warnf("csrf protection: missing origin and referer, method=%s, path=%s, ip=%s",
				c.Request.Method, c.Request.URL.Path, c.ClientIP())
			response.Error(c, response.ErrForbidden, "CSRF protection: missing origin header")
			c.Abort()
			return
		}

		valid := false

		if origin != "" {
			if strings.Contains(origin, host) || isValidOrigin(origin) {
				valid = true
			}
		}

		if !valid && referer != "" {
			if strings.Contains(referer, host) || isValidOrigin(referer) {
				valid = true
			}
		}

		if !valid {
			utils.Warnf("csrf protection: origin/referer mismatch, origin=%s, referer=%s, host=%s, path=%s, ip=%s",
				origin, referer, host, c.Request.URL.Path, c.ClientIP())
			response.Error(c, response.ErrForbidden, "CSRF protection: invalid origin")
			c.Abort()
			return
		}

		c.Next()
	}
}

func isValidOrigin(origin string) bool {
	allowedOrigins := config.GetStringSlice("cors.allowedOrigins")
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(origin, "://"+domain) || strings.HasSuffix(origin, "."+domain) {
				return true
			}
		}
	}
	return false
}
