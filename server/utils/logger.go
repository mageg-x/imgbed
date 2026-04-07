package utils

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logOnce   sync.Once          // 确保日志只初始化一次
	logLogger *zap.SugaredLogger // 全局日志实例
)

// InitLogger 初始化日志系统
// 根据运行模式选择不同的日志配置
// 参数：
//   - mode: 运行模式，"release"为生产模式，其他为开发模式
//
// 返回：
//   - error: 初始化失败时的错误
func InitLogger(mode string) error {
	var err error
	logOnce.Do(func() {
		var zapLogger *zap.Logger
		if mode == "release" {
			// 生产模式：使用JSON格式输出，性能优先
			zapLogger, err = zap.NewProduction()
		} else {
			// 开发模式：使用彩色控制台输出，便于调试
			cfg := zap.NewDevelopmentConfig()
			cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			cfg.OutputPaths = []string{"stdout"}
			cfg.ErrorOutputPaths = []string{"stderr"}
			// 禁用栈追踪
			cfg.EncoderConfig.StacktraceKey = ""
			zapLogger, err = cfg.Build()
		}
		if err != nil {
			return
		}
		logLogger = zapLogger.Sugar()
	})
	return err
}

// Debug 记录调试级别日志
func Debug(args ...interface{}) { logLogger.Debug(args...) }

// Debugf 记录格式化调试级别日志
func Debugf(template string, args ...interface{}) { logLogger.Debugf(template, args...) }

// Info 记录信息级别日志
func Info(args ...interface{}) { logLogger.Info(args...) }

// Infof 记录格式化信息级别日志
func Infof(template string, args ...interface{}) { logLogger.Infof(template, args...) }

// Warn 记录警告级别日志
func Warn(args ...interface{}) { logLogger.Warn(args...) }

// Warnf 记录格式化警告级别日志
func Warnf(template string, args ...interface{}) { logLogger.Warnf(template, args...) }

// Error 记录错误级别日志
func Error(args ...interface{}) { logLogger.Error(args...) }

// Errorf 记录格式化错误级别日志
func Errorf(template string, args ...interface{}) { logLogger.Errorf(template, args...) }

// Fatal 记录致命级别日志并退出程序
func Fatal(args ...interface{}) { logLogger.Fatal(args...); os.Exit(1) }

// Fatalf 记录格式化致命级别日志并退出程序
func Fatalf(template string, args ...interface{}) { logLogger.Fatalf(template, args...); os.Exit(1) }
