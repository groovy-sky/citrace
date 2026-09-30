package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ExecutionRecord struct {
	SchemaVersion       string            `json:"schemaVersion"`
	ID                  string            `json:"id"`
	Name                string            `json:"name,omitempty"`
	Provider            string            `json:"provider"`
	RequestedExecutable string            `json:"requestedExecutable"`
	ResolvedExecutable  string            `json:"resolvedExecutable"`
	Arguments           []string          `json:"arguments"`
	WorkingDirectory    string            `json:"workingDirectory"`
	PID                 int               `json:"pid,omitempty"`
	ProcessGroupID      int               `json:"processGroupId,omitempty"`
	StartedAt           time.Time         `json:"startedAt"`
	FinishedAt          *time.Time        `json:"finishedAt,omitempty"`
	DurationMS          *int64            `json:"durationMs,omitempty"`
	ExitCode            *int              `json:"exitCode"`
	Signal              string            `json:"signal,omitempty"`
	Error               string            `json:"error,omitempty"`
	StdoutLog           string            `json:"stdoutLog"`
	StderrLog           string            `json:"stderrLog"`
	ScriptSnapshot      string            `json:"scriptSnapshot,omitempty"`
	Warnings            []string          `json:"warnings,omitempty"`
	CI                  map[string]string `json:"ci,omitempty"`
}

func WriteRecord(directory string, record ExecutionRecord) error {
	temporaryPath := filepath.Join(directory, "execution.json.tmp")
	finalPath := filepath.Join(directory, "execution.json")
	file, err := os.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(record); err != nil {
		file.Close()
		os.Remove(temporaryPath)
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(temporaryPath)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(temporaryPath)
		return err
	}
	return os.Rename(temporaryPath, finalPath)
}
