package handlers

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

// SystemHandler 系统相关处理器
type SystemHandler struct {
	configService *services.ConfigService
	startTime     int64
}

// NewSystemHandler 创建系统处理器
func NewSystemHandler(configService *services.ConfigService) *SystemHandler {
	return &SystemHandler{
		configService: configService,
		startTime:     time.Now().Unix(),
	}
}

// GetSystemInfo 获取系统信息
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	// 获取主机信息
	hostInfo, _ := host.Info()

	// 获取应用信息
	appInfo := map[string]interface{}{
		"name":    "PanelBase",
		"version": "1.5.2",
		"uptime":  time.Now().Unix() - h.startTime,
	}

	// 获取Go运行时信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 获取当前工作目录
	wd, _ := os.Getwd()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"host": gin.H{
				"hostname":         hostInfo.Hostname,
				"os":               hostInfo.OS,
				"platform":         hostInfo.Platform,
				"platform_version": hostInfo.PlatformVersion,
				"kernel_version":   hostInfo.KernelVersion,
				"uptime":           hostInfo.Uptime,
			},
			"app": appInfo,
			"runtime": gin.H{
				"go_version":   runtime.Version(),
				"goroutines":   runtime.NumGoroutine(),
				"cpu_cores":    runtime.NumCPU(),
				"memory_alloc": m.Alloc,
				"memory_total": m.TotalAlloc,
				"memory_sys":   m.Sys,
				"working_dir":  wd,
			},
		},
	})
}

// GetSystemStatus 获取系统状态
func (h *SystemHandler) GetSystemStatus(c *gin.Context) {
	// 获取CPU信息
	cpuPercent, _ := cpu.Percent(time.Second, false)

	// 获取内存信息
	memInfo, _ := mem.VirtualMemory()

	// 获取磁盘信息
	diskInfo, _ := disk.Usage("/")

	// 获取当前负载
	loadAvg, _ := getLoadAverage()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"cpu": gin.H{
				"usage": cpuPercent,
			},
			"memory": gin.H{
				"total":     memInfo.Total,
				"used":      memInfo.Used,
				"available": memInfo.Available,
				"percent":   memInfo.UsedPercent,
			},
			"disk": gin.H{
				"total":     diskInfo.Total,
				"used":      diskInfo.Used,
				"available": diskInfo.Free,
				"percent":   diskInfo.UsedPercent,
			},
			"load": loadAvg,
		},
	})
}

// getLoadAverage 获取系统负载
func getLoadAverage() (map[string]float64, error) {
	loadStat, err := load.Avg()
	if err != nil {
		return map[string]float64{
			"load1":  0,
			"load5":  0,
			"load15": 0,
		}, err
	}

	return map[string]float64{
		"load1":  loadStat.Load1,
		"load5":  loadStat.Load5,
		"load15": loadStat.Load15,
	}, nil
}
