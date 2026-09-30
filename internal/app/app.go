package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/citrace/citrace-shell/internal/cli"
	"github.com/citrace/citrace-shell/internal/execution"
	"github.com/citrace/citrace-shell/internal/logging"
	"github.com/citrace/citrace-shell/internal/provider"
	"github.com/citrace/citrace-shell/internal/version"
)

func Run(arguments []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(arguments) == 1 && arguments[0] == "--version" {
		fmt.Fprintf(stdout, "citrace-shell %s\n", version.Value)
		return 0
	}
	if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h") {
		_, _ = cli.Parse([]string{"--help", "--", "unused"}, getenv, stdout)
		return 0
	}
	config, err := cli.Parse(arguments, getenv, stderr)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stderr, "citrace-shell: %v\n", err)
		}
		return 125
	}

	resolvedExecutable, err := execution.Resolve(config.Command)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintf(stderr, "citrace-shell: executable not found: %s\n", config.Command)
			return 127
		}
		fmt.Fprintf(stderr, "citrace-shell: resolve executable: %v\n", err)
		return 126
	}
	files, err := logging.NewFiles(config.LogDirectory, time.Now(), os.Getpid())
	if err != nil {
		fmt.Fprintf(stderr, "citrace-shell: %v\n", err)
		return 125
	}
	defer files.Close()

	startedAt := time.Now().UTC()
	workingDirectory, cwdError := os.Getwd()
	record := logging.ExecutionRecord{
		SchemaVersion:       "1.0",
		ID:                  files.ID,
		Name:                config.ExecutionName,
		RequestedExecutable: config.Command,
		ResolvedExecutable:  resolvedExecutable,
		Arguments:           logging.RedactArguments(config.Arguments),
		WorkingDirectory:    workingDirectory,
		StartedAt:           startedAt,
		StdoutLog:           "stdout.log",
		StderrLog:           "stderr.log",
	}
	selectedProvider := provider.Detect(getenv)
	record.Provider = selectedProvider.Name()
	record.CI = selectedProvider.Metadata()
	if cwdError != nil {
		record.Warnings = append(record.Warnings, "determine working directory: "+cwdError.Error())
	}
	trySnapshot(files, config.Arguments, &record)

	childStdout, childStderr := io.Writer(files.Stdout), io.Writer(files.Stderr)
	if !config.Quiet {
		childStdout = io.MultiWriter(stdout, files.Stdout)
		childStderr = io.MultiWriter(stderr, files.Stderr)
	}
	process, err := execution.Start(resolvedExecutable, config.Arguments, stdin, childStdout, childStderr)
	if err != nil {
		exitCode := 126
		record.ExitCode = &exitCode
		record.Error = err.Error()
		finishRecord(files.Directory, &record, startedAt)
		fmt.Fprintf(stderr, "citrace-shell: start executable: %v\n", err)
		return exitCode
	}
	record.PID = process.PID()
	record.ProcessGroupID = process.PID()
	if err := logging.WriteRecord(files.Directory, record); err != nil {
		_ = process.Signal(syscall.SIGTERM)
		_ = process.Wait()
		fmt.Fprintf(stderr, "citrace-shell: write initial record: %v\n", err)
		return 125
	}

	result := process.Wait()
	record.ExitCode = &result.ExitCode
	record.Signal = result.Signal
	if result.Error != nil && result.Signal == "" {
		record.Error = result.Error.Error()
	}
	if err := finishRecord(files.Directory, &record, startedAt); err != nil {
		fmt.Fprintf(stderr, "citrace-shell: write final record: %v\n", err)
	}
	return result.ExitCode
}

func trySnapshot(files *logging.Files, arguments []string, record *logging.ExecutionRecord) {
	if len(arguments) == 0 {
		return
	}
	candidate := arguments[len(arguments)-1]
	info, err := os.Stat(candidate)
	if err != nil || !info.Mode().IsRegular() {
		return
	}
	name, err := files.Snapshot(candidate, logging.MaxScriptSize)
	if err != nil {
		record.Warnings = append(record.Warnings, "snapshot script: "+err.Error())
		return
	}
	record.ScriptSnapshot = name
}

func finishRecord(directory string, record *logging.ExecutionRecord, startedAt time.Time) error {
	finishedAt := time.Now().UTC()
	duration := finishedAt.Sub(startedAt).Milliseconds()
	record.FinishedAt = &finishedAt
	record.DurationMS = &duration
	return logging.WriteRecord(directory, *record)
}
