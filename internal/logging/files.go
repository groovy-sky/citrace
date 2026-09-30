package logging

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const MaxScriptSize int64 = 1 << 20

type Files struct {
	ID        string
	Directory string
	Stdout    *os.File
	Stderr    *os.File
}

func NewFiles(root string, now time.Time, processID int) (*Files, error) {
	if err := os.MkdirAll(root, 0750); err != nil {
		return nil, fmt.Errorf("create log root: %w", err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect log root: %w", err)
	}
	if !rootInfo.IsDir() {
		return nil, errors.New("log root is not a directory")
	}

	id, err := executionID(now, processID)
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(root, id)
	if err := os.Mkdir(directory, 0750); err != nil {
		return nil, fmt.Errorf("create execution directory: %w", err)
	}
	stdout, err := exclusiveFile(filepath.Join(directory, "stdout.log"), 0640)
	if err != nil {
		return nil, fmt.Errorf("create stdout log: %w", err)
	}
	stderr, err := exclusiveFile(filepath.Join(directory, "stderr.log"), 0640)
	if err != nil {
		stdout.Close()
		return nil, fmt.Errorf("create stderr log: %w", err)
	}
	return &Files{ID: id, Directory: directory, Stdout: stdout, Stderr: stderr}, nil
}

func (files *Files) Close() error {
	return errors.Join(files.Stdout.Close(), files.Stderr.Close())
}

func (files *Files) Snapshot(path string, maximumSize int64) (string, error) {
	source, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("script is not a regular file")
	}
	if info.Size() > maximumSize {
		return "", fmt.Errorf("script exceeds %d byte limit", maximumSize)
	}
	destination, err := exclusiveFile(filepath.Join(files.Directory, "script.sh"), 0600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return "script.sh", nil
}

func exclusiveFile(path string, mode os.FileMode) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
}

func executionID(now time.Time, processID int) (string, error) {
	random := make([]byte, 3)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate execution ID: %w", err)
	}
	return fmt.Sprintf("%s-%d-%s", now.UTC().Format("20060102T150405.000000000Z"), processID, hex.EncodeToString(random)), nil
}
