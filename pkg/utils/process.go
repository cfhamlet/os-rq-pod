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
