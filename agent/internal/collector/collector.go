package collector

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	stdnet "net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"github.com/veona/agent/internal/buffer"
	"github.com/veona/agent/internal/config"
)

type Config struct {
	Global config.Config
}

type Collector struct {
	cfg      Config
	hostname string
}

func NewCollector(cfg Config) *Collector {
	hostname, err := host.Info()
	name := "unknown"
	if err == nil {
		name = hostname.Hostname
	}
	return &Collector{
		cfg:      cfg,
		hostname: name,
	}
}

// Run executes the collection workers
func (c *Collector) Run(ctx context.Context, buf *buffer.RingBuffer) {
	if c.cfg.Global.Collectors.CPU.Enabled {
		go c.collectCPU(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.CPU.Interval))
	}
	if c.cfg.Global.Collectors.Mem.Enabled {
		go c.collectMem(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.Mem.Interval))
	}
	if c.cfg.Global.Collectors.Disk.Enabled {
		go c.collectDisk(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.Disk.Interval))
	}
	if c.cfg.Global.Collectors.Swap.Enabled {
		go c.collectSwap(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.Swap.Interval))
	}
	if c.cfg.Global.Collectors.Load.Enabled {
		go c.collectLoad(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.Load.Interval))
	}
	if c.cfg.Global.Collectors.Net.Enabled {
		go c.collectNet(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.Net.Interval))
	}
	if c.cfg.Global.Collectors.ProcessStates.Enabled {
		go c.collectProcessStates(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.ProcessStates.Interval))
	}
	if c.cfg.Global.Collectors.Temperatures.Enabled {
		go c.collectTemperatures(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.Temperatures.Interval))
	}
	if c.cfg.Global.Collectors.Entropy.Enabled {
		go c.collectEntropy(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.Entropy.Interval))
	}
	if c.cfg.Global.Collectors.GPU.Enabled {
		go c.collectGPU(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.GPU.Interval))
	}
	if c.cfg.Global.Collectors.Battery.Enabled {
		go c.collectBattery(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.Battery.Interval))
	}
	if c.cfg.Global.Collectors.TimeSync.Enabled {
		go c.collectTimeSync(ctx, buf, config.ParseInterval(c.cfg.Global.Collectors.TimeSync.Interval))
	}

	// Internal Self-Monitoring (Agent resource usage)
	go c.collectInternal(ctx, buf, 1*time.Minute)

	<-ctx.Done()
	slog.Info("Collector workers stopped.")
}

func (c *Collector) collectCPU(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			if cpuPercents, err := cpu.Percent(0, false); err == nil && len(cpuPercents) > 0 {
				metrics["cpu_usage_percent"] = cpuPercents[0]
			}
			if cores, err := cpu.Counts(true); err == nil {
				metrics["cpu_core_count"] = cores
			}
			metrics["hostname"] = c.hostname

			buf.Push(buffer.MetricPayload{
				Timestamp: time.Now().Unix(),
				Metrics:   metrics,
			})
		}
	}
}

func (c *Collector) collectLoad(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			if loadStat, err := load.Avg(); err == nil {
				metrics["load_1"] = loadStat.Load1
				metrics["load_5"] = loadStat.Load5
				metrics["load_15"] = loadStat.Load15
			}
			metrics["hostname"] = c.hostname

			buf.Push(buffer.MetricPayload{
				Timestamp: time.Now().Unix(),
				Metrics:   metrics,
			})
		}
	}
}

func (c *Collector) collectMem(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			if vMem, err := mem.VirtualMemory(); err == nil {
				metrics["mem_total"] = vMem.Total
				metrics["mem_free"] = vMem.Free
				metrics["mem_available"] = vMem.Available
				metrics["mem_used_percent"] = vMem.UsedPercent
			}

			metrics["hostname"] = c.hostname

			buf.Push(buffer.MetricPayload{
				Timestamp: time.Now().Unix(),
				Metrics:   metrics,
			})
		}
	}
}

func (c *Collector) collectSwap(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			if sMem, err := mem.SwapMemory(); err == nil {
				metrics["swap_total"] = sMem.Total
				metrics["swap_free"] = sMem.Free
				// avoid div by zero safely
				if sMem.Total > 0 {
					metrics["swap_used_percent"] = (float64(sMem.Total-sMem.Free) / float64(sMem.Total)) * 100.0
				}
			}
			metrics["hostname"] = c.hostname

			buf.Push(buffer.MetricPayload{
				Timestamp: time.Now().Unix(),
				Metrics:   metrics,
			})
		}
	}
}

func (c *Collector) collectNet(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			if netIOCounters, err := net.IOCounters(false); err == nil && len(netIOCounters) > 0 {
				metrics["net_bytes_recv"] = netIOCounters[0].BytesRecv
				metrics["net_bytes_sent"] = netIOCounters[0].BytesSent
			}

			metrics["hostname"] = c.hostname

			buf.Push(buffer.MetricPayload{
				Timestamp: time.Now().Unix(),
				Metrics:   metrics,
			})
		}
	}
}

func (c *Collector) collectDisk(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Pre-compute exclude map
	excludeMap := make(map[string]bool)
	for _, fs := range c.cfg.Global.Collectors.Disk.ExcludeFS {
		excludeMap[fs] = true
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hostName := c.hostname

			if c.cfg.Global.Collectors.Disk.AutoDiscover {
				partitions, err := disk.Partitions(false)
				if err == nil {
					for _, p := range partitions {
						if excludeMap[p.Fstype] {
							continue
						}

						metrics := make(map[string]interface{})
						if hostName != "" {
							metrics["hostname"] = hostName
						}
						// Passing the path as a payload key to uniquely identify the disk
						metrics["mountpoint"] = p.Mountpoint

						// Add timeout to disk usage call
						usageChan := make(chan *disk.UsageStat, 1)
						go func(path string) {
							if usage, err := disk.Usage(path); err == nil {
								usageChan <- usage
							} else {
								usageChan <- nil
							}
						}(p.Mountpoint)

						select {
						case usage := <-usageChan:
							if usage != nil {
								metrics["disk_total"] = usage.Total
								metrics["disk_free"] = usage.Free
								metrics["disk_used_percent"] = usage.UsedPercent
							}
						case <-time.After(2 * time.Second):
							slog.Warn("Disk usage call timed out", "mountpoint", p.Mountpoint)
						}

						buf.Push(buffer.MetricPayload{
							Timestamp: time.Now().Unix(),
							Metrics:   metrics,
						})
					}
				}
			}
		}
	}
}

func (c *Collector) collectProcessStates(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			// Protect performance by adding a timeout to process listing
			procCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			procs, err := process.ProcessesWithContext(procCtx)
			cancel()
			if err == nil {
				stateCounts := map[string]int{
					"running":  0,
					"sleeping": 0,
					"stopped":  0,
					"zombie":   0,
					"idle":     0,
					"other":    0,
				}

				for _, p := range procs {
					status, err := p.Status()
					if err != nil || len(status) == 0 {
						continue
					}

					s := strings.ToLower(status[0])
					switch s {
					case "r", "running":
						stateCounts["running"]++
					case "s", "sleeping":
						stateCounts["sleeping"]++
					case "t", "stopped":
						stateCounts["stopped"]++
					case "z", "zombie":
						stateCounts["zombie"]++
					case "i", "idle":
						stateCounts["idle"]++
					default:
						stateCounts["other"]++
					}
				}
				metrics["process_count_running"] = stateCounts["running"]
				metrics["process_count_sleeping"] = stateCounts["sleeping"]
				metrics["process_count_stopped"] = stateCounts["stopped"]
				metrics["process_count_zombie"] = stateCounts["zombie"]
				metrics["process_count_idle"] = stateCounts["idle"]
				metrics["process_count_other"] = stateCounts["other"]
				metrics["process_count_total"] = len(procs)
			}

			metrics["hostname"] = c.hostname

			buf.Push(buffer.MetricPayload{
				Timestamp: time.Now().Unix(),
				Metrics:   metrics,
			})
		}
	}
}

func (c *Collector) collectTemperatures(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			temps, err := host.SensorsTemperatures()
			if err == nil {
				for _, t := range temps {
					key := "temp_" + strings.ReplaceAll(strings.ToLower(t.SensorKey), " ", "_")
					metrics[key] = t.Temperature
				}
			}

			metrics["hostname"] = c.hostname

			if len(metrics) > 1 {
				buf.Push(buffer.MetricPayload{
					Timestamp: time.Now().Unix(),
					Metrics:   metrics,
				})
			}
		}
	}
}

func (c *Collector) collectEntropy(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			if runtime.GOOS == "linux" {
				b, err := os.ReadFile("/proc/sys/kernel/random/entropy_avail")
				if err == nil {
					valStr := strings.TrimSpace(string(b))
					if val, err := strconv.Atoi(valStr); err == nil {
						metrics["system_entropy"] = val
					}
				}
			}

			metrics["hostname"] = c.hostname

			if len(metrics) > 1 {
				buf.Push(buffer.MetricPayload{
					Timestamp: time.Now().Unix(),
					Metrics:   metrics,
				})
			}
		}
	}
}

func (c *Collector) collectGPU(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			// Try to run nvidia-smi with timeout
			cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			out, err := exec.CommandContext(cmdCtx, "nvidia-smi", "--query-gpu=utilization.gpu,utilization.memory,memory.used,memory.total", "--format=csv,noheader,nounits").Output()
			cancel()
			if err == nil {
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				for i, line := range lines {
					parts := strings.Split(line, ",")
					if len(parts) >= 4 {
						gpuUtil, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
						memUtil, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
						memUsed, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
						memTotal, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)

						prefix := fmt.Sprintf("gpu_%d_", i)
						metrics[prefix+"utilization_percent"] = gpuUtil
						metrics[prefix+"mem_utilization_percent"] = memUtil
						metrics[prefix+"mem_used_mb"] = memUsed
						metrics[prefix+"mem_total_mb"] = memTotal
					}
				}
			}

			if len(metrics) > 0 {
				metrics["hostname"] = c.hostname
				buf.Push(buffer.MetricPayload{
					Timestamp: time.Now().Unix(),
					Metrics:   metrics,
				})
			}
		}
	}
}

func (c *Collector) collectBattery(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})
			if runtime.GOOS == "linux" {
				b, err := os.ReadFile("/sys/class/power_supply/BAT0/capacity")
				if err == nil {
					valStr := strings.TrimSpace(string(b))
					if val, err := strconv.Atoi(valStr); err == nil {
						metrics["battery_capacity_percent"] = val
					}
				}
			} else if runtime.GOOS == "windows" {
				cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				out, err := exec.CommandContext(cmdCtx, "wmic", "path", "Win32_Battery", "get", "EstimatedChargeRemaining").Output()
				cancel()
				if err == nil {
					lines := strings.Split(strings.TrimSpace(string(out)), "\n")
					if len(lines) > 1 {
						valStr := strings.TrimSpace(lines[1])
						if val, err := strconv.Atoi(valStr); err == nil {
							metrics["battery_capacity_percent"] = val
						}
					}
				}
			}

			if len(metrics) > 0 {
				metrics["hostname"] = c.hostname
				buf.Push(buffer.MetricPayload{
					Timestamp: time.Now().Unix(),
					Metrics:   metrics,
				})
			}
		}
	}
}

func (c *Collector) collectTimeSync(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			// Simple SNTP request to pool.ntp.org
			conn, err := stdnet.Dial("udp", "pool.ntp.org:123")
			if err == nil {
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				req := make([]byte, 48)
				req[0] = 0x1B // client mode, version 3

				if _, err := conn.Write(req); err == nil {
					resp := make([]byte, 48)
					if _, err := conn.Read(resp); err == nil {
						// Seconds since 1900
						secs := binary.BigEndian.Uint32(resp[40:44])
						frac := binary.BigEndian.Uint32(resp[44:48])

						// 2208988800 = seconds between 1900 and 1970
						unixSecs := int64(secs) - 2208988800
						unixNano := (int64(frac) * 1e9) >> 32
						ntpTime := time.Unix(unixSecs, unixNano)

						drift := time.Since(ntpTime).Milliseconds()
						metrics["ntp_drift_ms"] = float64(drift)
					}
				}
				conn.Close()
			}

			if len(metrics) > 0 {
				metrics["hostname"] = c.hostname
				buf.Push(buffer.MetricPayload{
					Timestamp: time.Now().Unix(),
					Metrics:   metrics,
				})
			}
		}
	}
}

func (c *Collector) collectInternal(ctx context.Context, buf *buffer.RingBuffer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			metrics["agent_mem_alloc_bytes"] = m.Alloc
			metrics["agent_mem_sys_bytes"] = m.Sys
			metrics["agent_goroutines"] = runtime.NumGoroutine()

			metrics["hostname"] = c.hostname

			buf.Push(buffer.MetricPayload{
				Timestamp: time.Now().Unix(),
				Metrics:   metrics,
			})
		}
	}
}
