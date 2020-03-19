package utils

import (
	"os"

	"github.com/shirou/gopsutil/process"
)

// NewProcess TODO
func NewProcess() (*process.Process, error) {
	pid := os.Getpid()
	return process.NewProcess(int32(pid))
}

// MemoryInfo TODO
func MemoryInfo(p *process.Process) *process.MemoryInfoStat {
	m, _ := p.MemoryInfo()
	return m
}

// CPUPercent TODO
func CPUPercent(p *process.Process) float64 {
	c, _ := p.CPUPercent()
	return c
}
