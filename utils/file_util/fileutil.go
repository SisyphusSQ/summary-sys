package file_util

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Exists returns true when path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates a directory tree when missing.
func EnsureDir(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("ensure directory %s: %w", path, err)
	}

	return nil
}

// ReadJSON reads JSON file into out.
func ReadJSON(path string, out any) error {
	if out == nil {
		return fmt.Errorf("output target cannot be nil")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read json file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal json file %s: %w", path, err)
	}

	return nil
}

// WriteJSON writes value as indented JSON.
func WriteJSON(path string, value any, perm os.FileMode) error {
	if perm == 0 {
		perm = 0o644
	}
	parentDir := filepath.Dir(path)
	if parentDir != "." {
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return fmt.Errorf("create parent directory %s: %w", parentDir, err)
		}
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json for %s: %w", path, err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("write json file %s: %w", path, err)
	}

	return nil
}

// CopyFile copies file from srcPath to dstPath.
func CopyFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source file %s: %w", srcPath, err)
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("source file is not regular: %s", srcPath)
	}

	parentDir := filepath.Dir(dstPath)
	if parentDir != "." {
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return fmt.Errorf("create parent directory %s: %w", parentDir, err)
		}
	}

	dstFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open destination file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy %s to %s: %w", srcPath, dstPath, err)
	}
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("sync destination file %s: %w", dstPath, err)
	}

	return nil
}
