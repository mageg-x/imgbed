package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/imgbed/server/config"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/storage"
	"github.com/imgbed/server/utils"
)

const (
	ScheduleStrategyRoundRobin = "round_robin"
	ScheduleStrategyRandom     = "random"
	ScheduleStrategyPriority   = "priority"
)

var sensitiveFields = []string{
	"token", "secret", "password", "apiKey", "api_key", "accessKey", "access_key",
	"secretKey", "secret_key", "privateKey", "private_key", "credential",
}

var channelSecretKey = []byte("imgbed-channel-secret-key-32byte")

var (
	channelServiceInstance *ChannelService
	channelServiceOnce     sync.Once
)

type ChannelService struct {
	drivers       map[string]storage.StorageDriver
	mu            sync.RWMutex
	roundRobinIdx int
	roundRobinMu  sync.Mutex
}

func NewChannelService() *ChannelService {
	channelServiceOnce.Do(func() {
		channelServiceInstance = &ChannelService{
			drivers: make(map[string]storage.StorageDriver),
		}
	})
	return channelServiceInstance
}

func getScheduleStrategy() string {
	strategy := config.GetString("schedule.strategy")
	if strategy == "" {
		return ScheduleStrategyPriority
	}
	return strategy
}

func encryptValue(plaintext string) (string, error) {
	block, err := aes.NewCipher(channelSecretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := cryptorand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "enc:" + hex.EncodeToString(ciphertext), nil
}

func decryptValue(ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, "enc:") {
		return ciphertext, nil
	}

	data, err := hex.DecodeString(ciphertext[4:])
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(channelSecretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func isSensitiveField(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, field := range sensitiveFields {
		if strings.Contains(lowerKey, strings.ToLower(field)) {
			return true
		}
	}
	return false
}

func encryptConfig(configMap map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range configMap {
		if isSensitiveField(k) {
			if strVal, ok := v.(string); ok && strVal != "" {
				encrypted, err := encryptValue(strVal)
				if err != nil {
					return nil, fmt.Errorf("encrypt field %s failed: %w", k, err)
				}
				result[k] = encrypted
				continue
			}
		}
		result[k] = v
	}
	return result, nil
}

func decryptConfig(configMap map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range configMap {
		if isSensitiveField(k) {
			if strVal, ok := v.(string); ok && strings.HasPrefix(strVal, "enc:") {
				decrypted, err := decryptValue(strVal)
				if err != nil {
					utils.Warnf("decrypt config: decrypt field %s failed, error=%v", k, err)
					result[k] = v
					continue
				}
				result[k] = decrypted
				continue
			}
		}
		result[k] = v
	}
	return result, nil
}

// DecryptChannelConfig 解密通道配置
func DecryptChannelConfig(configJSON string) (map[string]interface{}, error) {
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return nil, err
	}
	return decryptConfig(configMap)
}

func (s *ChannelService) CreateChannel(ctx context.Context, name string, channelType string, config map[string]interface{}, quota model.QuotaConfig, rateLimit model.RateLimitConfig) (*model.Channel, error) {
	channelID := utils.GenerateID()

	encryptedConfig, err := encryptConfig(config)
	if err != nil {
		utils.Errorf("create channel: encrypt config failed, name=%s, type=%s, error=%v", name, channelType, err)
		return nil, fmt.Errorf("encrypt config failed: %w", err)
	}

	configJSON, err := json.Marshal(encryptedConfig)
	if err != nil {
		utils.Errorf("create channel: marshal config failed, name=%s, type=%s, error=%v", name, channelType, err)
		return nil, fmt.Errorf("marshal config failed: %w", err)
	}

	channel := &model.Channel{
		ID:                channelID,
		Name:              name,
		Type:              channelType,
		Config:            string(configJSON),
		Enabled:           true,
		Status:            "healthy",
		QuotaEnabled:      quota.Enabled,
		QuotaLimit:        int64(quota.LimitGB) * 1024 * 1024 * 1024,
		QuotaThreshold:    quota.Threshold,
		DailyUploadLimit:  rateLimit.DailyUploadLimit,
		HourlyUploadLimit: rateLimit.HourlyUploadLimit,
		MinIntervalMs:     rateLimit.MinIntervalMs,
		Weight:            100,
		LastUsedAt:        time.Now(),
	}

	if err := database.DB.Create(channel).Error; err != nil {
		utils.Errorf("create channel: save to database failed, name=%s, type=%s, error=%v", name, channelType, err)
		return nil, fmt.Errorf("create channel failed: %w", err)
	}

	utils.Infof("create channel: success, id=%s, name=%s, type=%s", channelID, name, channelType)
	return channel, nil
}

func (s *ChannelService) UpdateChannel(ctx context.Context, channelID string, name string, config map[string]interface{}, quota model.QuotaConfig, rateLimit model.RateLimitConfig) error {
	var channel model.Channel
	if err := database.DB.Where("id = ?", channelID).First(&channel).Error; err != nil {
		utils.Errorf("update channel: channel not found, channelID=%s, error=%v", channelID, err)
		return fmt.Errorf("channel not found")
	}

	encryptedConfig, err := encryptConfig(config)
	if err != nil {
		utils.Errorf("update channel: encrypt config failed, channelID=%s, error=%v", channelID, err)
		return fmt.Errorf("encrypt config failed: %w", err)
	}

	configJSON, err := json.Marshal(encryptedConfig)
	if err != nil {
		utils.Errorf("update channel: marshal config failed, channelID=%s, error=%v", channelID, err)
		return fmt.Errorf("marshal config failed: %w", err)
	}

	updates := map[string]interface{}{
		"name":                name,
		"config":              string(configJSON),
		"quota_enabled":       quota.Enabled,
		"quota_limit":         int64(quota.LimitGB) * 1024 * 1024 * 1024,
		"quota_threshold":     quota.Threshold,
		"daily_upload_limit":  rateLimit.DailyUploadLimit,
		"hourly_upload_limit": rateLimit.HourlyUploadLimit,
		"min_interval_ms":     rateLimit.MinIntervalMs,
	}

	s.mu.Lock()
	delete(s.drivers, channelID)
	s.mu.Unlock()

	if err := database.DB.Model(&channel).Updates(updates).Error; err != nil {
		utils.Errorf("update channel: database update failed, channelID=%s, error=%v", channelID, err)
		return err
	}

	utils.Infof("update channel: success, channelID=%s, name=%s", channelID, name)
	return nil
}

// DeleteChannel 删除存储通道
// 参数：
//   - ctx: 上下文
//   - channelID: 通道ID
//
// 返回：
//   - error: 删除过程中的错误
func (s *ChannelService) DeleteChannel(ctx context.Context, channelID string) error {
	// 检查通道是否有关联的文件
	var count int64
	database.DB.Model(&model.File{}).Where("channel_id = ?", channelID).Count(&count)
	if count > 0 {
		utils.Warnf("delete channel: channel has files, channelID=%s, fileCount=%d", channelID, count)
		return fmt.Errorf("cannot delete channel with files")
	}

	// 清除缓存的驱动实例
	s.mu.Lock()
	delete(s.drivers, channelID)
	s.mu.Unlock()

	// 删除通道记录
	if err := database.DB.Where("id = ?", channelID).Delete(&model.Channel{}).Error; err != nil {
		utils.Errorf("delete channel: database delete failed, channelID=%s, error=%v", channelID, err)
		return err
	}

	utils.Infof("delete channel: success, channelID=%s", channelID)
	return nil
}

// EnableChannel 启用或禁用存储通道
// 参数：
//   - ctx: 上下文
//   - channelID: 通道ID
//   - enabled: 是否启用
//
// 返回：
//   - error: 操作过程中的错误
func (s *ChannelService) EnableChannel(ctx context.Context, channelID string, enabled bool) error {
	if err := database.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("enabled", enabled).Error; err != nil {
		utils.Errorf("enable channel: database update failed, channelID=%s, enabled=%v, error=%v", channelID, enabled, err)
		return err
	}
	utils.Infof("enable channel: success, channelID=%s, enabled=%v", channelID, enabled)
	return nil
}

func (s *ChannelService) GetChannel(ctx context.Context, channelID string) (*model.Channel, error) {
	var channel model.Channel
	if err := database.DB.Where("id = ?", channelID).First(&channel).Error; err != nil {
		utils.Errorf("get channel: channel not found, channelID=%s, error=%v", channelID, err)
		return nil, fmt.Errorf("channel not found")
	}
	return &channel, nil
}

func (s *ChannelService) GetChannelWithDecryptedConfig(ctx context.Context, channelID string) (*model.Channel, map[string]interface{}, error) {
	channel, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, nil, err
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(channel.Config), &configMap); err != nil {
		utils.Errorf("get channel with decrypted config: unmarshal failed, channelID=%s, error=%v", channelID, err)
		return channel, nil, fmt.Errorf("parse config failed: %w", err)
	}

	decryptedConfig, err := decryptConfig(configMap)
	if err != nil {
		utils.Warnf("get channel with decrypted config: decrypt failed, channelID=%s, error=%v", channelID, err)
		return channel, configMap, nil
	}

	return channel, decryptedConfig, nil
}

// ListChannels 获取所有存储通道列表
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - []model.Channel: 通道列表
//   - error: 获取过程中的错误
func (s *ChannelService) ListChannels(ctx context.Context) ([]model.Channel, error) {
	var channels []model.Channel
	if err := database.DB.Order("created_at DESC").Find(&channels).Error; err != nil {
		utils.Errorf("list channels: query failed, error=%v", err)
		return nil, err
	}
	return channels, nil
}

func (s *ChannelService) GetDriver(channelID string) (storage.StorageDriver, error) {
	s.mu.RLock()
	if driver, ok := s.drivers[channelID]; ok {
		s.mu.RUnlock()
		return driver, nil
	}
	s.mu.RUnlock()

	var channel model.Channel
	if err := database.DB.Where("id = ?", channelID).First(&channel).Error; err != nil {
		utils.Errorf("get driver: channel not found, channelID=%s, error=%v", channelID, err)
		return nil, fmt.Errorf("channel not found")
	}

	if !channel.Enabled {
		utils.Warnf("get driver: channel is disabled, channelID=%s", channelID)
		return nil, fmt.Errorf("channel is disabled")
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(channel.Config), &configMap); err != nil {
		utils.Errorf("get driver: parse config failed, channelID=%s, error=%v", channelID, err)
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	decryptedConfig, err := decryptConfig(configMap)
	if err != nil {
		utils.Warnf("get driver: decrypt config failed, channelID=%s, error=%v", channelID, err)
		decryptedConfig = configMap
	}

	driverConfig := &storage.ChannelConfig{
		ID:             channel.ID,
		Name:           channel.Name,
		Type:           storage.StorageType(channel.Type),
		Enabled:        channel.Enabled,
		Config:         decryptedConfig,
		QuotaEnabled:   channel.QuotaEnabled,
		QuotaLimit:     channel.QuotaLimit,
		QuotaThreshold: channel.QuotaThreshold,
		DailyLimit:     channel.DailyUploadLimit,
		HourlyLimit:    channel.HourlyUploadLimit,
		MinIntervalMs:  channel.MinIntervalMs,
	}

	driver, err := storage.CreateDriver(driverConfig)
	if err != nil {
		utils.Errorf("get driver: create driver failed, channelID=%s, error=%v", channelID, err)
		return nil, err
	}

	s.mu.Lock()
	s.drivers[channelID] = driver
	s.mu.Unlock()

	return driver, nil
}

// SelectChannel 根据负载均衡算法选择最合适的通道
func (s *ChannelService) SelectChannel(ctx context.Context, fileSize int64) (string, error) {
	candidates := s.getAvailableChannels()
	if len(candidates) == 0 {
		utils.Warnf("select channel: no available channels after filtering")
		return "", fmt.Errorf("no available channels")
	}

	strategy := getScheduleStrategy()
	var selectedID string

	switch strategy {
	case ScheduleStrategyRoundRobin:
		selectedID = s.selectByRoundRobin(candidates)
	case ScheduleStrategyRandom:
		selectedID = s.selectByRandom(candidates)
	default:
		selectedID = s.selectByPriority(candidates)
	}

	utils.Debugf("select channel: selected channelID=%s, strategy=%s", selectedID, strategy)
	return selectedID, nil
}

func (s *ChannelService) getAvailableChannels() []*model.Channel {
	var channels []model.Channel
	if err := database.DB.Where("enabled = ?", true).Find(&channels).Error; err != nil {
		utils.Errorf("get available channels: query failed, error=%v", err)
		return nil
	}

	candidates := make([]*model.Channel, 0)
	for i := range channels {
		ch := &channels[i]

		if ch.Status == "cooldown" {
			continue
		}

		if ch.QuotaEnabled && ch.QuotaLimit > 0 {
			usagePercent := float64(ch.UsedSpace) / float64(ch.QuotaLimit) * 100
			if usagePercent >= float64(ch.QuotaThreshold) {
				continue
			}
		}

		if ch.HourlyUploadLimit > 0 && ch.HourlyUploads >= ch.HourlyUploadLimit {
			continue
		}

		if ch.DailyUploadLimit > 0 && ch.DailyUploads >= ch.DailyUploadLimit {
			continue
		}

		if ch.MinIntervalMs > 0 {
			elapsed := time.Since(ch.LastUsedAt).Milliseconds()
			if elapsed < int64(ch.MinIntervalMs) {
				continue
			}
		}

		candidates = append(candidates, ch)
	}

	return candidates
}

func (s *ChannelService) selectByRoundRobin(channels []*model.Channel) string {
	s.roundRobinMu.Lock()
	defer s.roundRobinMu.Unlock()

	if len(channels) == 0 {
		return ""
	}

	idx := s.roundRobinIdx % len(channels)
	s.roundRobinIdx++
	return channels[idx].ID
}

func (s *ChannelService) selectByRandom(channels []*model.Channel) string {
	if len(channels) == 0 {
		return ""
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return channels[r.Intn(len(channels))].ID
}

func (s *ChannelService) selectByPriority(channels []*model.Channel) string {
	if len(channels) == 0 {
		return ""
	}

	type scoredChannel struct {
		channel *model.Channel
		score   float64
	}

	scored := make([]scoredChannel, len(channels))
	for i, ch := range channels {
		score := float64(ch.Weight)

		if ch.QuotaEnabled && ch.QuotaLimit > 0 {
			usagePercent := float64(ch.UsedSpace) / float64(ch.QuotaLimit)
			score *= (1 - usagePercent)
		}

		if ch.HourlyUploadLimit > 0 {
			remainingPercent := 1 - float64(ch.HourlyUploads)/float64(ch.HourlyUploadLimit)
			score *= remainingPercent
		}

		scored[i] = scoredChannel{channel: ch, score: score}
	}

	var best *scoredChannel
	for i := range scored {
		if best == nil || scored[i].score > best.score {
			best = &scored[i]
		}
	}

	if best == nil {
		return channels[0].ID
	}
	return best.channel.ID
}

// UpdateUsage 更新通道的使用量统计
// 参数：
//   - channelID: 通道ID
//   - deltaSize: 变化的存储大小（字节，正数表示增加，负数表示减少）
func (s *ChannelService) UpdateUsage(channelID string, deltaSize int64) {
	// 更新使用量、上传计数器和最后使用时间
	if err := database.DB.Model(&model.Channel{}).Where("id = ?", channelID).Updates(map[string]interface{}{
		"used_space":     database.DB.Raw("used_space + ?", deltaSize),
		"hourly_uploads": database.DB.Raw("hourly_uploads + 1"),
		"daily_uploads":  database.DB.Raw("daily_uploads + 1"),
		"last_used_at":   time.Now(),
	}).Error; err != nil {
		utils.Errorf("update usage: database update failed, channelID=%s, deltaSize=%d, error=%v", channelID, deltaSize, err)
	}
}

// GetChannelStatus 获取通道的详细状态
// 参数：
//   - ctx: 上下文
//   - channelID: 通道ID
//
// 返回：
//   - *model.ChannelStatus: 通道状态对象
//   - error: 获取过程中的错误
func (s *ChannelService) GetChannelStatus(ctx context.Context, channelID string) (*model.ChannelStatus, error) {
	channel, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	// 统计通道关联的文件数量
	var fileCount int64
	database.DB.Model(&model.File{}).Where("channel_id = ?", channelID).Count(&fileCount)

	// 计算存储使用百分比
	usagePercent := 0
	if channel.QuotaLimit > 0 {
		usagePercent = int(float64(channel.UsedSpace) / float64(channel.QuotaLimit) * 100)
	}

	return &model.ChannelStatus{
		ID:           channel.ID,
		Name:         channel.Name,
		Type:         channel.Type,
		Enabled:      channel.Enabled,
		Status:       channel.Status,
		UsedSpace:    channel.UsedSpace,
		TotalSpace:   channel.QuotaLimit,
		UsagePercent: usagePercent,
		FileCount:    int(fileCount),
	}, nil
}

// GetAllChannelStatus 获取所有通道的状态列表
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - []model.ChannelStatus: 所有通道的状态列表
//   - error: 获取过程中的错误
func (s *ChannelService) GetAllChannelStatus(ctx context.Context) ([]model.ChannelStatus, error) {
	channels, err := s.ListChannels(ctx)
	if err != nil {
		utils.Errorf("get all channel status: list channels failed, error=%v", err)
		return nil, err
	}

	statuses := make([]model.ChannelStatus, len(channels))
	for i, ch := range channels {
		// 统计每个通道的文件数量
		var fileCount int64
		database.DB.Model(&model.File{}).Where("channel_id = ?", ch.ID).Count(&fileCount)

		// 计算存储使用百分比
		usagePercent := 0
		if ch.QuotaLimit > 0 {
			usagePercent = int(float64(ch.UsedSpace) / float64(ch.QuotaLimit) * 100)
		}

		statuses[i] = model.ChannelStatus{
			ID:           ch.ID,
			Name:         ch.Name,
			Type:         ch.Type,
			Enabled:      ch.Enabled,
			Status:       ch.Status,
			UsedSpace:    ch.UsedSpace,
			TotalSpace:   ch.QuotaLimit,
			UsagePercent: usagePercent,
			FileCount:    int(fileCount),
		}
	}

	return statuses, nil
}

// HealthCheck 检查指定通道的健康状态
// 参数：
//   - ctx: 上下文
//   - channelID: 通道ID
//
// 返回：
//   - error: 健康检查结果（nil表示健康）
func (s *ChannelService) HealthCheck(ctx context.Context, channelID string) error {
	// 获取通道的存储驱动
	driver, err := s.GetDriver(channelID)
	if err != nil {
		// 更新通道状态为错误
		database.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("status", "error")
		utils.Errorf("health check: get driver failed, channelID=%s, error=%v", channelID, err)
		return err
	}

	// 执行驱动的健康检查
	if err := driver.HealthCheck(ctx); err != nil {
		// 更新通道状态为错误
		database.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("status", "error")
		utils.Errorf("health check: driver health check failed, channelID=%s, error=%v", channelID, err)
		return err
	}

	// 更新通道状态为健康
	if err := database.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("status", "healthy").Error; err != nil {
		utils.Warnf("health check: update status to healthy failed, channelID=%s, error=%v", channelID, err)
	}

	return nil
}

// HealthCheckAll 检查所有通道的健康状态
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - map[string]error: 通道ID到健康检查结果的映射
func (s *ChannelService) HealthCheckAll(ctx context.Context) map[string]error {
	channels, _ := s.ListChannels(ctx)

	results := make(map[string]error)
	for _, ch := range channels {
		results[ch.ID] = s.HealthCheck(ctx, ch.ID)
	}

	return results
}

// ResetHourlyCounters 重置所有通道的小时计数器
func (s *ChannelService) ResetHourlyCounters() {
	if err := database.DB.Model(&model.Channel{}).Where("1 = 1").Update("hourly_uploads", 0).Error; err != nil {
		utils.Errorf("reset hourly counters: database update failed, error=%v", err)
	}
}

// ResetDailyCounters 重置所有通道的日计数器
func (s *ChannelService) ResetDailyCounters() {
	if err := database.DB.Model(&model.Channel{}).Where("1 = 1").Update("daily_uploads", 0).Error; err != nil {
		utils.Errorf("reset daily counters: database update failed, error=%v", err)
	}
}

// SetChannelWeight 设置通道权重
// 参数：
//   - channelID: 通道ID
//   - weight: 权重值 (0-100)
//
// 返回：
//   - error: 设置过程中的错误
func (s *ChannelService) SetChannelWeight(channelID string, weight int) error {
	// 验证权重范围
	if weight < 0 {
		weight = 0
	}
	if weight > 100 {
		weight = 100
	}

	// 更新通道权重
	if err := database.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("weight", weight).Error; err != nil {
		utils.Errorf("set channel weight: update failed, channelID=%s, weight=%d, error=%v", channelID, weight, err)
		return fmt.Errorf("set weight failed: %w", err)
	}

	utils.Infof("set channel weight: success, channelID=%s, weight=%d", channelID, weight)
	return nil
}

// SelectChannelExcluding 选择通道，排除指定的失败渠道
// 用于重试时切换到不同的渠道
func (s *ChannelService) SelectChannelExcluding(ctx context.Context, fileSize int64, excludeChannels map[string]bool) (string, error) {
	var channels []model.Channel
	if err := database.DB.Where("enabled = ?", true).Find(&channels).Error; err != nil {
		utils.Errorf("select channel excluding: query enabled channels failed, error=%v", err)
		return "", fmt.Errorf("no available channels")
	}

	if len(channels) == 0 {
		utils.Warnf("select channel excluding: no channels configured")
		return "", fmt.Errorf("no channels configured")
	}

	type candidate struct {
		channel *model.Channel
		score   float64
	}

	candidates := make([]candidate, 0)

	for i := range channels {
		ch := &channels[i]

		if excludeChannels[ch.ID] {
			continue
		}

		if ch.Status == "cooldown" {
			continue
		}

		if ch.QuotaEnabled && ch.QuotaLimit > 0 {
			usagePercent := float64(ch.UsedSpace) / float64(ch.QuotaLimit) * 100
			if usagePercent >= float64(ch.QuotaThreshold) {
				continue
			}
		}

		if ch.HourlyUploadLimit > 0 && ch.HourlyUploads >= ch.HourlyUploadLimit {
			continue
		}

		if ch.DailyUploadLimit > 0 && ch.DailyUploads >= ch.DailyUploadLimit {
			continue
		}

		if ch.MinIntervalMs > 0 {
			elapsed := time.Since(ch.LastUsedAt).Milliseconds()
			if elapsed < int64(ch.MinIntervalMs) {
				continue
			}
		}

		score := float64(ch.Weight)

		if ch.QuotaEnabled && ch.QuotaLimit > 0 {
			usagePercent := float64(ch.UsedSpace) / float64(ch.QuotaLimit)
			score *= (1 - usagePercent)
		}

		if ch.HourlyUploadLimit > 0 {
			remainingPercent := 1 - float64(ch.HourlyUploads)/float64(ch.HourlyUploadLimit)
			score *= remainingPercent
		}

		candidates = append(candidates, candidate{channel: ch, score: score})
	}

	if len(candidates) == 0 {
		utils.Warnf("select channel excluding: no available channels after filtering, excluded=%d", len(excludeChannels))
		return "", fmt.Errorf("no available channels")
	}

	var best *candidate
	for i := range candidates {
		if best == nil || candidates[i].score > best.score {
			best = &candidates[i]
		}
	}

	utils.Debugf("select channel excluding: selected channelID=%s, score=%f, excluded=%d", best.channel.ID, best.score, len(excludeChannels))
	return best.channel.ID, nil
}

func (s *ChannelService) MarkChannelCooldown(ctx context.Context, channelID string) error {
	var channel model.Channel
	if err := database.DB.Where("id = ?", channelID).First(&channel).Error; err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	cooldownMinutes := channel.CooldownMinutes
	if cooldownMinutes <= 0 {
		cooldownMinutes = 60
	}

	cooldownUntil := time.Now().Add(time.Duration(cooldownMinutes) * time.Minute)

	if err := database.DB.Model(&model.Channel{}).Where("id = ?", channelID).Updates(map[string]interface{}{
		"status":     "cooldown",
		"updated_at": time.Now(),
	}).Error; err != nil {
		utils.Errorf("mark channel cooldown: update failed, channelID=%s, error=%v", channelID, err)
		return err
	}

	utils.Warnf("mark channel cooldown: channel marked as cooldown, channelID=%s, cooldownMinutes=%d, cooldownUntil=%s",
		channelID, cooldownMinutes, cooldownUntil.Format("2006-01-02 15:04:05"))

	s.mu.Lock()
	delete(s.drivers, channelID)
	s.mu.Unlock()

	return nil
}

func (s *ChannelService) RecoverChannelFromCooldown(ctx context.Context, channelID string) error {
	if err := database.DB.Model(&model.Channel{}).Where("id = ? AND status = ?", channelID, "cooldown").Updates(map[string]interface{}{
		"status":     "healthy",
		"updated_at": time.Now(),
	}).Error; err != nil {
		utils.Errorf("recover channel from cooldown: update failed, channelID=%s, error=%v", channelID, err)
		return err
	}

	utils.Infof("recover channel from cooldown: channel recovered, channelID=%s", channelID)

	s.mu.Lock()
	delete(s.drivers, channelID)
	s.mu.Unlock()

	return nil
}

func (s *ChannelService) StartCooldownRecovery() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			s.recoverExpiredCooldowns()
		}
	}()
}

func (s *ChannelService) recoverExpiredCooldowns() {
	var channels []model.Channel
	if err := database.DB.Where("status = ? AND updated_at < ?", "cooldown", time.Now().Add(-time.Hour)).Find(&channels).Error; err != nil {
		utils.Warnf("recover expired cooldowns: query failed, error=%v", err)
		return
	}

	for _, ch := range channels {
		cooldownMinutes := ch.CooldownMinutes
		if cooldownMinutes <= 0 {
			cooldownMinutes = 60
		}

		cooldownExpiry := ch.UpdatedAt.Add(time.Duration(cooldownMinutes) * time.Minute)
		if time.Now().After(cooldownExpiry) {
			if err := s.RecoverChannelFromCooldown(context.Background(), ch.ID); err != nil {
				utils.Warnf("recover expired cooldowns: recover failed, channelID=%s, error=%v", ch.ID, err)
			}
		}
	}
}
