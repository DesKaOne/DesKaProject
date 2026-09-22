package p2p

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), fmt.Sprintf("\"%d\"", pid))
	}
	return exec.Command("kill", "-0", strconv.Itoa(pid)).Run() == nil
}
