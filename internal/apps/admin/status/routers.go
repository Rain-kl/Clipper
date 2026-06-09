/*
Copyright 2025 linux.do
Modified by Arctel.net, 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package status 提供系统状态查询接口
package status

import (
	"fmt"
	"math"
	"net/http"
	"runtime"
	"time"

	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

// startTime 记录服务启动时间
var startTime = time.Now()

const (
	hoursInDay      = 24
	minutesInHour   = 60
	secondsInMinute = 60
	nanosPerSecond  = 1e9
	binaryKB        = 2
	binaryMB        = 3
	binaryGB        = 4
	valueThreshold  = 10 // 格式化时区分整数显示的阈值
)

// SystemStatusResponse 系统状态响应结构体
type SystemStatusResponse struct {
	Uptime       string `json:"uptime"`
	NumGoroutine int    `json:"num_goroutine"`
	Alloc        string `json:"alloc"`
	TotalAlloc   string `json:"total_alloc"`
	Sys          string `json:"sys"`
	Lookups      uint64 `json:"lookups"`
	Mallocs      uint64 `json:"mallocs"`
	Frees        uint64 `json:"frees"`
	HeapAlloc    string `json:"heap_alloc"`
	HeapSys      string `json:"heap_sys"`
	HeapIdle     string `json:"heap_idle"`
	HeapInuse    string `json:"heap_inuse"`
	HeapReleased string `json:"heap_released"`
	HeapObjects  uint64 `json:"heap_objects"`
	StackInuse   string `json:"stack_inuse"`
	StackSys     string `json:"stack_sys"`
	MSpanInuse   string `json:"mspan_inuse"`
	MSpanSys     string `json:"mspan_sys"`
	MCacheInuse  string `json:"mcache_inuse"`
	MCacheSys    string `json:"mcache_sys"`
	BuckHashSys  string `json:"buck_hash_sys"`
	GCSys        string `json:"gc_sys"`
	OtherSys     string `json:"other_sys"`
	NextGC       string `json:"next_gc"`
	LastGCTime   string `json:"last_gc_time"`
	PauseTotalNs string `json:"pause_total_ns"`
	LastPause    string `json:"last_pause"`
	NumGC        uint32 `json:"num_gc"`
}

// formatBytes 格式化字节大小
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	value := float64(bytes) / float64(div)
	var suffix string
	switch exp {
	case binaryKB:
		suffix = "KiB"
	case binaryMB:
		suffix = "MiB"
	case binaryGB:
		suffix = "GiB"
	default:
		suffix = "TiB"
	}

	// 格式化规则：
	// - 如果是整数（如 16, 73, 105, 986, 112）：
	//   - 如果 >= 10，则格式化为 "%.0f" (e.g. "16 KiB")
	//   - 如果 < 10，则格式化为 "%.1f" (e.g. "9.0 KiB")
	// - 如果不是整数（如 5.8, 9.1, 7.6, 4.8）：格式化为 "%.1f"
	if value == math.Trunc(value) {
		if value >= valueThreshold {
			return fmt.Sprintf("%.0f %s", value, suffix)
		}
		return fmt.Sprintf("%.1f %s", value, suffix)
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}

// formatDuration 格式化时间持续时间
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / hoursInDay
	hours := int(d.Hours()) % hoursInDay
	minutes := int(d.Minutes()) % minutesInHour
	seconds := int(d.Seconds()) % secondsInMinute

	var res string
	if days > 0 {
		res += fmt.Sprintf("%d天", days)
	}
	if hours > 0 {
		res += fmt.Sprintf("%d小时", hours)
	}
	if minutes > 0 {
		res += fmt.Sprintf("%d分钟", minutes)
	}
	if seconds > 0 || res == "" {
		res += fmt.Sprintf("%d秒钟", seconds)
	}
	return res
}

// GetSystemStatus 获取系统状态信息
// @Summary 获取系统状态信息
// @Description 获取后端服务运行状态、Goroutine、内存指标等详细统计数据，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=status.SystemStatusResponse} "获取成功"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Router /api/v1/admin/status [get]
func GetSystemStatus(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := formatDuration(time.Since(startTime))
	numGoroutine := runtime.NumGoroutine()

	var lastGCTime string
	if m.LastGC > 0 {
		lastGCTime = formatDuration(time.Since(time.Unix(0, int64(m.LastGC))))
	} else {
		lastGCTime = "无"
	}

	var lastPause string
	if m.NumGC > 0 {
		lastPause = fmt.Sprintf("%.3fs", float64(m.PauseNs[(m.NumGC-1)%256])/nanosPerSecond)
	} else {
		lastPause = "0.000s"
	}

	res := SystemStatusResponse{
		Uptime:       uptime,
		NumGoroutine: numGoroutine,
		Alloc:        formatBytes(m.Alloc),
		TotalAlloc:   formatBytes(m.TotalAlloc),
		Sys:          formatBytes(m.Sys),
		Lookups:      m.Lookups,
		Mallocs:      m.Mallocs,
		Frees:        m.Frees,
		HeapAlloc:    formatBytes(m.HeapAlloc),
		HeapSys:      formatBytes(m.HeapSys),
		HeapIdle:     formatBytes(m.HeapIdle),
		HeapInuse:    formatBytes(m.HeapInuse),
		HeapReleased: formatBytes(m.HeapReleased),
		HeapObjects:  m.HeapObjects,
		StackInuse:   formatBytes(m.StackInuse),
		StackSys:     formatBytes(m.StackSys),
		MSpanInuse:   formatBytes(m.MSpanInuse),
		MSpanSys:     formatBytes(m.MSpanSys),
		MCacheInuse:  formatBytes(m.MCacheInuse),
		MCacheSys:    formatBytes(m.MCacheSys),
		BuckHashSys:  formatBytes(m.BuckHashSys),
		GCSys:        formatBytes(m.GCSys),
		OtherSys:     formatBytes(m.OtherSys),
		NextGC:       formatBytes(m.NextGC),
		LastGCTime:   lastGCTime,
		PauseTotalNs: fmt.Sprintf("%.1fs", float64(m.PauseTotalNs)/nanosPerSecond),
		LastPause:    lastPause,
		NumGC:        m.NumGC,
	}

	c.JSON(http.StatusOK, util.OK(res))
}
