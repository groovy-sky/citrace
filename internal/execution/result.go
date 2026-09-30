package execution

import (
	"fmt"
	"strings"
	"syscall"
)

type Result struct {
	ExitCode int
	Signal   string
	Error    error
}

func resultFromWait(commandError error, status syscall.WaitStatus) Result {
	if status.Signaled() {
		signal := status.Signal()
		return Result{
			ExitCode: 128 + int(signal),
			Signal:   signalName(signal),
			Error:    commandError,
		}
	}
	return Result{ExitCode: status.ExitStatus(), Error: commandError}
}

func signalName(signal syscall.Signal) string {
	known := map[syscall.Signal]string{
		syscall.SIGHUP:  "SIGHUP",
		syscall.SIGINT:  "SIGINT",
		syscall.SIGQUIT: "SIGQUIT",
		syscall.SIGKILL: "SIGKILL",
		syscall.SIGTERM: "SIGTERM",
	}
	if name, ok := known[signal]; ok {
		return name
	}
	return "SIG" + strings.ToUpper(fmt.Sprint(int(signal)))
}
