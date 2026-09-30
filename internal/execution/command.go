package execution

import (
	"io"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	command    *exec.Cmd
	signals    chan os.Signal
	signalDone chan struct{}
}

func Resolve(executable string) (string, error) {
	return exec.LookPath(executable)
}

func Start(executable string, arguments []string, stdin io.Reader, stdout, stderr io.Writer) (*Process, error) {
	command := exec.Command(executable, arguments...)
	command.Stdout = stdout
	command.Stderr = stderr
	command.Stdin = stdin
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	process := &Process{command: command}
	process.prepareSignalForwarding()
	if err := command.Start(); err != nil {
		process.stopSignalForwarding()
		return nil, err
	}
	process.forwardSignals()
	return process, nil
}

func (process *Process) PID() int {
	return process.command.Process.Pid
}

func (process *Process) Signal(signal syscall.Signal) error {
	return syscall.Kill(-process.PID(), signal)
}
