package metrics

import (
	"os/exec"
	"syscall"
)

func readDriverGPU() (GPU, bool)      { return GPU{}, false }
func hideCommandWindow(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
