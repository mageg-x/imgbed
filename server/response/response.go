package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/utils"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
	Pages    int   `json:"pages"`
}

type PaginatedResponse struct {
	List       interface{} `json:"list"`
	Pagination Pagination  `json:"pagination"`
}

const (
	SuccessCode = 0

	ErrBadRequest      = 10001
	ErrUnauthorized    = 10002
	ErrForbidden       = 10003
	ErrNotFound        = 10004
	ErrInternal        = 10005
	ErrValidation      = 10006
	ErrTooManyRequests = 10007
	ErrFileTooLarge    = 10008
	ErrFileType        = 10009
	ErrRateLimitAnon   = 10010
	ErrInvalidParam    = 10011
	ErrDuplicateName   = 10012

	ErrChannelNotFound    = 20001
	ErrChannelDisabled    = 20002
	ErrChannelConfig      = 20003
	ErrChannelTestFailed  = 20004
	ErrChannelQuotaFull   = 20005
	ErrChannelRateLimit   = 20006
	ErrNoAvailableChannel = 20007
	ErrChannelUnavailable = 20008

	ErrCodeTokenDisabled = 30001
	ErrCodeTokenExpired  = 30002
	ErrCodeTokenInvalid  = 30003
	ErrCodePermission    = 30004

	ErrCodeUploadFailed   = 40001
	ErrCodeDownloadFailed = 40002
	ErrCodeDeleteFailed   = 40003
	ErrCodeCompressFailed = 40004
	ErrCodeChannelError   = 40005

	ErrConfigNotFound = 50001
	ErrConfigInvalid  = 50002
)

var errorMessages = map[int]string{
	ErrBadRequest:         "Bad request",
	ErrUnauthorized:       "Unauthorized",
	ErrForbidden:          "Forbidden",
	ErrNotFound:           "Not found",
	ErrInternal:           "Internal server error",
	ErrValidation:         "Validation error",
	ErrTooManyRequests:    "Too many requests",
	ErrFileTooLarge:       "File too large",
	ErrFileType:           "Unsupported file type",
	ErrRateLimitAnon:      "Anonymous upload rate limit exceeded",
	ErrInvalidParam:       "Invalid parameter",
	ErrDuplicateName:      "Duplicate name",
	ErrChannelNotFound:    "Channel not found",
	ErrChannelDisabled:    "Channel is disabled",
	ErrChannelConfig:      "Channel configuration error",
	ErrChannelTestFailed:  "Channel test failed",
	ErrChannelQuotaFull:   "Channel quota is full",
	ErrChannelRateLimit:   "Channel rate limit exceeded",
	ErrNoAvailableChannel: "No available channel",
	ErrChannelUnavailable: "Channel is unavailable",
	ErrCodeTokenDisabled:  "Token is disabled",
	ErrCodeTokenExpired:   "Token has expired",
	ErrCodeTokenInvalid:   "Invalid token",
	ErrCodePermission:     "Permission denied",
	ErrCodeUploadFailed:   "Upload failed",
	ErrCodeDownloadFailed: "Download failed",
	ErrCodeDeleteFailed:   "Delete failed",
	ErrCodeCompressFailed: "Image compression failed",
	ErrCodeChannelError:   "Channel error",
	ErrConfigNotFound:     "Configuration not found",
	ErrConfigInvalid:      "Invalid configuration",
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	if message == "" {
		if msg, ok := errorMessages[code]; ok {
			message = msg
		}
	}

	status := http.StatusBadRequest
	switch code {
	case ErrUnauthorized, ErrCodeTokenInvalid, ErrCodeTokenExpired, ErrCodeTokenDisabled:
		status = http.StatusUnauthorized
	case ErrForbidden, ErrCodePermission:
		status = http.StatusForbidden
	case ErrNotFound:
		status = http.StatusNotFound
	case ErrInternal:
		status = http.StatusInternalServerError
	case ErrTooManyRequests, ErrRateLimitAnon:
		status = http.StatusTooManyRequests
	}

	utils.Debugf("response error: code=%d, message=%s, status=%d", code, message, status)

	c.JSON(status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, ErrBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, ErrUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, ErrForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, ErrNotFound, message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, ErrInternal, message)
}

func ValidationError(c *gin.Context, message string) {
	Error(c, ErrValidation, message)
}

func TooManyRequests(c *gin.Context, message string) {
	Error(c, ErrTooManyRequests, message)
}

func FileTooLarge(c *gin.Context, message string) {
	Error(c, ErrFileTooLarge, message)
}

func FileTypeError(c *gin.Context, message string) {
	Error(c, ErrFileType, message)
}

func TokenError(c *gin.Context, code int, message string) {
	Error(c, code, message)
}

func UploadError(c *gin.Context, message string) {
	Error(c, ErrCodeUploadFailed, message)
}

func DownloadError(c *gin.Context, message string) {
	Error(c, ErrCodeDownloadFailed, message)
}

func DeleteError(c *gin.Context, message string) {
	Error(c, ErrCodeDeleteFailed, message)
}

func ChannelError(c *gin.Context, code int, message string) {
	Error(c, code, message)
}

func ChannelNotFound(c *gin.Context, message string) {
	Error(c, ErrChannelNotFound, message)
}

func ChannelDisabled(c *gin.Context, message string) {
	Error(c, ErrChannelDisabled, message)
}

func NoAvailableChannel(c *gin.Context, message string) {
	Error(c, ErrNoAvailableChannel, message)
}

func ConfigError(c *gin.Context, code int, message string) {
	Error(c, code, message)
}

func PaginatedSuccess(c *gin.Context, list interface{}, page, pageSize int, total int64) {
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}

	Success(c, PaginatedResponse{
		List: list,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			Pages:    pages,
		},
	})
}
