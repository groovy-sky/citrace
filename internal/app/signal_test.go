package app

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSignalHelper(t *testing.T) {
	if os.Getenv("CITRACE_TEST_SIGNAL_HELPER") != "1" {
		return
	}
	exitCode := Run(
		[]string{"--", "/bin/sh", "-c", "trap 'exit 0' TERM; sleep 30 & child=$!; echo $child; wait $child"},
		os.Getenv,
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)
	os.Exit(exitCode)
}

func TestSignalIsForwardedToChildProcessGroup(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestSignalHelper$")
	command.Env = append(os.Environ(), "CITRACE_TEST_SIGNAL_HELPER=1", "CITRACE_LOG_DIR="+t.TempDir())
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("read descendant PID: %v; stderr: %s", err, stderr.String())
	}
	descendantPID, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatalf("parse descendant PID %q: %v", line, err)
	}
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	select {
	case err := <-waited:
		if err != nil {
			t.Fatalf("launcher did not preserve trapped child status: %v; stderr: %s", err, stderr.String())
		}
	case <-time.After(3 * time.Second):
		_ = command.Process.Kill()
		<-waited
		t.Fatal("launcher did not forward SIGTERM within three seconds")
	}
	if err := syscall.Kill(descendantPID, 0); err == nil || err != syscall.ESRCH {
		t.Fatalf("descendant process %d remains: %v", descendantPID, err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", fmt.Sprintf("%q", stderr.String()))
	}
}
