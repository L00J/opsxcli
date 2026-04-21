package sys

import (
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// collectProcesses 收集进程信息，同时返回系统总进程数
func (dc *DataCollector) collectProcesses(isFirst bool) ([]*ProcessInfo, int) {
	processes, err := process.Processes()
	if err != nil || len(processes) == 0 {
		return []*ProcessInfo{}, -1
	}

	// 记录系统总进程数（避免重复查询）
	totalProcessCount := len(processes)

	procInfos := make([]*ProcessInfo, 0, 200)

	// 首次收集时只处理50个进程，加快首次加载
	maxProcs := 200
	if isFirst {
		maxProcs = 50
	}
	if len(processes) > maxProcs {
		processes = processes[:maxProcs]
	}

	semaphore := make(chan struct{}, 20)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, p := range processes {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(proc *process.Process) {
			defer wg.Done()
			defer func() { <-semaphore }()

			name, err := proc.Name()
			if err != nil {
				return
			}

			var cpuPercent float64
			var memPercent float32
			var status []string
			var username, cmdline string
			var threads int32
			var createTime int64

			cpuPercent, _ = proc.CPUPercent()
			memPercent, _ = proc.MemoryPercent()
			status, _ = proc.Status()
			username, _ = proc.Username()
			cmdline, _ = proc.Cmdline()
			threads, _ = proc.NumThreads()
			createTime, _ = proc.CreateTime()

			times, _ := proc.Times()
			cpuTime := 0.0
			if times != nil {
				cpuTime = times.User + times.System
			}

			memInfo, _ := proc.MemoryInfo()
			memRSS := uint64(0)
			memVMS := uint64(0)
			if memInfo != nil {
				memRSS = memInfo.RSS
				memVMS = memInfo.VMS
			}

			diskRead, diskWrite := uint64(0), uint64(0)
			diskReadRate, diskWriteRate := 0.0, 0.0

			ioStat, ioErr := proc.IOCounters()
			if ioErr != nil {
				if runtime.GOOS == "darwin" {
					diskRead, diskWrite = 0, 0
				}
			} else if ioStat != nil {
				diskRead = ioStat.ReadBytes
				diskWrite = ioStat.WriteBytes

				mu.Lock()
				if lastIO, ok := dc.lastProcIO[proc.Pid]; ok {
					deltaTime := time.Since(lastIO.Timestamp).Seconds()
					if deltaTime > 0 && deltaTime < 10 {
						diskReadRate = float64(diskRead-lastIO.ReadBytes) / deltaTime
						diskWriteRate = float64(diskWrite-lastIO.WriteBytes) / deltaTime
						if diskReadRate < 0 {
							diskReadRate = 0
						}
						if diskWriteRate < 0 {
							diskWriteRate = 0
						}
					}
				}
				dc.lastProcIO[proc.Pid] = &ProcessIO{
					ReadBytes:  diskRead,
					WriteBytes: diskWrite,
					Timestamp:  time.Now(),
				}
				mu.Unlock()
			}

			runTime := time.Duration(0)
			if createTime > 0 {
				runTime = time.Since(time.Unix(createTime, 0))
			}

			procInfo := &ProcessInfo{
				PID:           proc.Pid,
				Name:          name,
				CPU:           cpuPercent,
				CPUTime:       cpuTime,
				Mem:           float32(memPercent),
				MemRSS:        memRSS,
				MemVMS:        memVMS,
				Status:        strings.Join(status, ","),
				User:          username,
				Command:       cmdline,
				Threads:       threads,
				CreateTime:    createTime,
				RunTime:       runTime,
				DiskRead:      diskRead,
				DiskWrite:     diskWrite,
				DiskReadRate:  diskReadRate,
				DiskWriteRate: diskWriteRate,
			}

			mu.Lock()
			procInfos = append(procInfos, procInfo)
			mu.Unlock()
		}(p)
	}

	wg.Wait()

	// 按CPU使用率排序
	sort.Slice(procInfos, func(i, j int) bool {
		return procInfos[i].CPU > procInfos[j].CPU
	})

	// 只保留前150个
	if len(procInfos) > 150 {
		procInfos = procInfos[:150]
	}

	return procInfos, totalProcessCount
}
