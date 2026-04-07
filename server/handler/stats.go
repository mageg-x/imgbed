package handler

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/database"
	"github.com/imgbed/server/model"
	"github.com/imgbed/server/response"
)

type StatsHandler struct{}

var (
	statsCache      = make(map[string]interface{})
	statsCacheTime  = make(map[string]time.Time)
	statsCacheMutex sync.RWMutex
	cacheDuration   = 5 * time.Minute
	cleanupInterval = 10 * time.Minute
)

func init() {
	go startCacheCleanup()
}

func startCacheCleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		cleanExpiredCache()
	}
}

func cleanExpiredCache() {
	statsCacheMutex.Lock()
	defer statsCacheMutex.Unlock()

	now := time.Now()
	for key, cacheTime := range statsCacheTime {
		if now.Sub(cacheTime) > cacheDuration {
			delete(statsCache, key)
			delete(statsCacheTime, key)
		}
	}
}

func NewStatsHandler() *StatsHandler {
	return &StatsHandler{}
}

func getFromCache(key string) (interface{}, bool) {
	statsCacheMutex.RLock()
	defer statsCacheMutex.RUnlock()
	if data, ok := statsCache[key]; ok {
		if time.Since(statsCacheTime[key]) < cacheDuration {
			return data, true
		}
	}
	return nil, false
}

func setCache(key string, data interface{}) {
	statsCacheMutex.Lock()
	defer statsCacheMutex.Unlock()
	statsCache[key] = data
	statsCacheTime[key] = time.Now()
}

// Overview 获取统计概览
// @Summary 获取统计概览
// @Description 获取系统上传统计概览数据
// @Tags 统计
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Router /stats/overview [get]
func (h *StatsHandler) Overview(c *gin.Context) {
	cacheKey := "stats_overview"
	forceRefresh := c.Query("refresh") == "true"
	if data, ok := getFromCache(cacheKey); ok && !forceRefresh {
		response.Success(c, data)
		return
	}

	var totalFiles int64
	var todayUploads int64
	var totalSuccess int64
	var totalFailed int64

	database.DB.Model(&model.File{}).Count(&totalFiles)

	today := time.Now().Format("2006-01-02")
	database.DB.Model(&model.File{}).Where("date(created_at) = ?", today).Count(&todayUploads)

	database.DB.Model(&model.FileAccess{}).Where("access_type = ?", "upload_success").Count(&totalSuccess)
	database.DB.Model(&model.FileAccess{}).Where("access_type = ?", "upload_failed").Count(&totalFailed)

	totalUploads := totalSuccess + totalFailed

	var successRate float64
	if totalUploads > 0 {
		successRate = float64(totalSuccess) / float64(totalUploads) * 100
	}

	data := gin.H{
		"totalFiles":   totalFiles,
		"totalUploads": totalUploads,
		"todayUploads": todayUploads,
		"totalSuccess": totalSuccess,
		"totalFailed":  totalFailed,
		"successRate":  successRate,
	}

	setCache(cacheKey, data)
	response.Success(c, data)
}

func (h *StatsHandler) Channels(c *gin.Context) {
	cacheKey := "stats_channels"
	if data, ok := getFromCache(cacheKey); ok {
		response.Success(c, data)
		return
	}

	type ChannelStat struct {
		ChannelID    string  `json:"channelId"`
		ChannelName  string  `json:"channelName"`
		ChannelType  string  `json:"channelType"`
		TotalUploads int64   `json:"totalUploads"`
		SuccessCount int64   `json:"successCount"`
		FailedCount  int64   `json:"failedCount"`
		SuccessRate  float64 `json:"successRate"`
		LastUsedAt   int64   `json:"lastUsedAt"`
	}

	var channels []model.Channel
	database.DB.Where("enabled = ?", true).Find(&channels)

	channelIDs := make([]string, len(channels))
	channelMap := make(map[string]*model.Channel)
	for i, ch := range channels {
		channelIDs[i] = ch.ID
		channelMap[ch.ID] = &channels[i]
	}

	type FileStat struct {
		ChannelID  string
		Count      int64
		LastUsedAt time.Time
	}
	var fileStats []FileStat
	database.DB.Model(&model.File{}).
		Select("channel_id, COUNT(*) as count, MAX(created_at) as last_used_at").
		Where("channel_id IN ?", channelIDs).
		Group("channel_id").
		Scan(&fileStats)

	fileStatMap := make(map[string]FileStat)
	for _, fs := range fileStats {
		fileStatMap[fs.ChannelID] = fs
	}

	type AccessStat struct {
		ChannelID  string
		AccessType string
		Count      int64
	}
	var accessStats []AccessStat
	database.DB.Table("file_accesses").
		Select("files.channel_id, file_accesses.access_type, COUNT(*) as count").
		Joins("JOIN files ON file_accesses.file_id = files.id").
		Where("files.channel_id IN ?", channelIDs).
		Group("files.channel_id, file_accesses.access_type").
		Scan(&accessStats)

	accessStatMap := make(map[string]map[string]int64)
	for _, as := range accessStats {
		if accessStatMap[as.ChannelID] == nil {
			accessStatMap[as.ChannelID] = make(map[string]int64)
		}
		accessStatMap[as.ChannelID][as.AccessType] = as.Count
	}

	channelStats := make([]ChannelStat, 0)
	for _, ch := range channels {
		fs := fileStatMap[ch.ID]
		as := accessStatMap[ch.ID]

		var successCount, failedCount int64
		if as != nil {
			successCount = as["upload_success"]
			failedCount = as["upload_failed"]
		}

		var successRate float64
		totalAttempts := successCount + failedCount
		if totalAttempts > 0 {
			successRate = float64(successCount) / float64(totalAttempts) * 100
		} else if fs.Count > 0 {
			successRate = 100.0
		}

		channelStats = append(channelStats, ChannelStat{
			ChannelID:    ch.ID,
			ChannelName:  ch.Name,
			ChannelType:  ch.Type,
			TotalUploads: fs.Count,
			SuccessCount: successCount,
			FailedCount:  failedCount,
			SuccessRate:  successRate,
			LastUsedAt:   fs.LastUsedAt.Unix(),
		})
	}

	data := gin.H{"items": channelStats}
	setCache(cacheKey, data)
	response.Success(c, data)
}

func (h *StatsHandler) Trend(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	cacheKey := "stats_trend_" + strconv.Itoa(days)
	if data, ok := getFromCache(cacheKey); ok {
		response.Success(c, data)
		return
	}

	type DailyTrend struct {
		Date    string  `json:"date"`
		Uploads int64   `json:"uploads"`
		Success int64   `json:"success"`
		Failed  int64   `json:"failed"`
		Rate    float64 `json:"rate"`
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	type FileDailyStat struct {
		Date  string
		Count int64
	}
	var fileStats []FileDailyStat
	database.DB.Table("files").
		Select("date(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("date(created_at)").
		Order("date ASC").
		Scan(&fileStats)

	type AccessDailyStat struct {
		Date       string
		AccessType string
		Count      int64
	}
	var accessStats []AccessDailyStat
	database.DB.Table("file_accesses").
		Select("date(access_at) as date, access_type, COUNT(*) as count").
		Where("access_at >= ?", startDate).
		Group("date(access_at), access_type").
		Scan(&accessStats)

	accessMap := make(map[string]map[string]int64)
	for _, as := range accessStats {
		if accessMap[as.Date] == nil {
			accessMap[as.Date] = make(map[string]int64)
		}
		accessMap[as.Date][as.AccessType] = as.Count
	}

	trends := make([]DailyTrend, 0, len(fileStats))
	for _, fs := range fileStats {
		as := accessMap[fs.Date]
		var successCount, failedCount int64
		if as != nil {
			successCount = as["upload_success"]
			failedCount = as["upload_failed"]
		}

		var rate float64
		total := successCount + failedCount
		if total > 0 {
			rate = float64(successCount) / float64(total) * 100
		} else {
			rate = 100.0
		}

		trends = append(trends, DailyTrend{
			Date:    fs.Date,
			Uploads: fs.Count,
			Success: successCount,
			Failed:  failedCount,
			Rate:    rate,
		})
	}

	data := gin.H{"items": trends}
	setCache(cacheKey, data)
	response.Success(c, data)
}

func (h *StatsHandler) Weekly(c *gin.Context) {
	cacheKey := "stats_weekly"
	if data, ok := getFromCache(cacheKey); ok {
		response.Success(c, data)
		return
	}

	type WeekStat struct {
		WeekStart string `json:"weekStart"`
		WeekEnd   string `json:"weekEnd"`
		Uploads   int64  `json:"uploads"`
		Success   int64  `json:"success"`
		Failed    int64  `json:"failed"`
	}

	var weeklyStats []WeekStat

	now := time.Now()
	for i := 0; i < 12; i++ {
		weekEnd := now.AddDate(0, 0, -i*7)
		weekStart := weekEnd.AddDate(0, 0, -6)

		var uploads int64
		database.DB.Model(&model.File{}).
			Where("created_at >= ? AND created_at <= ?",
				weekStart.Format("2006-01-02")+" 00:00:00",
				weekEnd.Format("2006-01-02")+" 23:59:59").
			Count(&uploads)

		var successCount, failedCount int64
		database.DB.Table("file_accesses").
			Where("access_at >= ? AND access_at <= ? AND access_type = ?",
				weekStart.Format("2006-01-02")+" 00:00:00",
				weekEnd.Format("2006-01-02")+" 23:59:59",
				"upload_success").
			Count(&successCount)
		database.DB.Table("file_accesses").
			Where("access_at >= ? AND access_at <= ? AND access_type = ?",
				weekStart.Format("2006-01-02")+" 00:00:00",
				weekEnd.Format("2006-01-02")+" 23:59:59",
				"upload_failed").
			Count(&failedCount)

		weeklyStats = append(weeklyStats, WeekStat{
			WeekStart: weekStart.Format("2006-01-02"),
			WeekEnd:   weekEnd.Format("2006-01-02"),
			Uploads:   uploads,
			Success:   successCount,
			Failed:    failedCount,
		})
	}

	data := gin.H{"items": weeklyStats}
	setCache(cacheKey, data)
	response.Success(c, data)
}
