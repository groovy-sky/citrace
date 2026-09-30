package execution

import (
	"syscall"
	"testing"
)

func TestSignalResult(t *testing.T) {
	result := resultFromWait(nil, syscall.WaitStatus(syscall.SIGTERM))
	if result.ExitCode != 143 || result.Signal != "SIGTERM" {
		t.Fatalf("unexpected signal result: %#v", result)
	}
}
