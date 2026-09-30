//go:build linux

package execution

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func (process *Process) Wait() Result {
	waitError := process.command.Wait()
	process.stopSignalForwarding()

	var status syscall.WaitStatus
	if process.command.ProcessState != nil {
		status, _ = process.command.ProcessState.Sys().(syscall.WaitStatus)
	}
	if waitError != nil {
		var exitError *exec.ExitError
		if !errors.As(waitError, &exitError) {
			return Result{ExitCode: 125, Error: waitError}
		}
	}
	return resultFromWait(nil, status)
}

func (process *Process) prepareSignalForwarding() {
	process.signals = make(chan os.Signal, 1)
	process.signalDone = make(chan struct{})
	signal.Notify(process.signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
}

func (process *Process) forwardSignals() {
	go func() {
		defer close(process.signalDone)
		for received := range process.signals {
			if forwarded, ok := received.(syscall.Signal); ok {
				_ = process.Signal(forwarded)
			}
		}
	}()
}

func (process *Process) stopSignalForwarding() {
	if process.signals == nil {
		return
	}
	signal.Stop(process.signals)
	close(process.signals)
	if process.command.Process != nil {
		<-process.signalDone
	}
	process.signals = nil
}
