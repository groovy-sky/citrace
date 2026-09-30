package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/citrace/citrace-shell/internal/logging"
)

func TestRunCapturesStreamsAndPreservesExitCode(t *testing.T) {
	logRoot := t.TempDir()
	environment := map[string]string{
		"CITRACE_LOG_DIR": logRoot,
		"GITHUB_ACTIONS":  "true",
		"GITHUB_JOB":      "test",
	}
	getenv := func(name string) string { return environment[name] }
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run(
		[]string{"--execution-name", "integration", "--", "/bin/sh", "-c", "read value; echo out:$value; echo err >&2; exit 7", "--token", "secret"},
		getenv,
		bytes.NewBufferString("input\n"),
		&stdout,
		&stderr,
	)
	if exitCode != 7 {
		t.Fatalf("got exit code %d; stderr: %s", exitCode, stderr.String())
	}
	if stdout.String() != "out:input\n" || stderr.String() != "err\n" {
		t.Fatalf("unexpected console output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	directory := onlyExecutionDirectory(t, logRoot)
	assertFileContents(t, filepath.Join(directory, "stdout.log"), "out:input\n")
	assertFileContents(t, filepath.Join(directory, "stderr.log"), "err\n")
	recordData, err := os.ReadFile(filepath.Join(directory, "execution.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record logging.ExecutionRecord
	if err := json.Unmarshal(recordData, &record); err != nil {
		t.Fatal(err)
	}
	if record.ExitCode == nil || *record.ExitCode != 7 || record.Provider != "github-actions" {
		t.Fatalf("unexpected record result: %#v", record)
	}
	if record.Arguments[len(record.Arguments)-1] != "[REDACTED]" {
		t.Fatalf("secret argument was not redacted: %#v", record.Arguments)
	}
	if record.PID == 0 || record.ProcessGroupID != record.PID || record.FinishedAt == nil {
		t.Fatalf("incomplete process metadata: %#v", record)
	}
}

func TestRunQuietStoresWithoutMirroring(t *testing.T) {
	logRoot := t.TempDir()
	getenv := func(name string) string {
		if name == "CITRACE_LOG_DIR" {
			return logRoot
		}
		return ""
	}
	var stdout bytes.Buffer
	exitCode := Run([]string{"--quiet", "--", "/bin/echo", "stored"}, getenv, bytes.NewReader(nil), &stdout, &bytes.Buffer{})
	if exitCode != 0 || stdout.Len() != 0 {
		t.Fatalf("quiet run returned %d with output %q", exitCode, stdout.String())
	}
	assertFileContents(t, filepath.Join(onlyExecutionDirectory(t, logRoot), "stdout.log"), "stored\n")
}

func onlyExecutionDirectory(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("unexpected execution directories: %#v", entries)
	}
	return filepath.Join(root, entries[0].Name())
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s contains %q, want %q", path, data, want)
	}
}
