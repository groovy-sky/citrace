package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilesAndAtomicRecord(t *testing.T) {
	files, err := NewFiles(t.TempDir(), time.Unix(0, 0), 42)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	record := ExecutionRecord{SchemaVersion: "1.0", ID: files.ID, ExitCode: nil}
	if err := WriteRecord(files.Directory, record); err != nil {
		t.Fatal(err)
	}
	record.Provider = "local"
	if err := WriteRecord(files.Directory, record); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(files.Directory, "execution.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded ExecutionRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Provider != "local" {
		t.Fatalf("unexpected record: %#v", decoded)
	}
	if _, err := os.Stat(filepath.Join(files.Directory, "execution.json.tmp")); !os.IsNotExist(err) {
		t.Fatalf("temporary record remains: %v", err)
	}
}

func TestSnapshotIsBounded(t *testing.T) {
	files, err := NewFiles(t.TempDir(), time.Now(), os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	script := filepath.Join(t.TempDir(), "step.sh")
	if err := os.WriteFile(script, []byte("echo hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if name, err := files.Snapshot(script, MaxScriptSize); err != nil || name != "script.sh" {
		t.Fatalf("snapshot failed: %q %v", name, err)
	}
}
