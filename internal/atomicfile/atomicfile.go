// Package atomicfile writes files so that a crash or kill mid-write can never
// leave them truncated: data goes to a temp file in the same directory, which
// is then renamed over the target. It is shared by every writer of
// credential stores and third-party tool configs in this module.
package atomicfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// BackupSuffix is appended to a file's path for the copy WriteWithBackup keeps.
const BackupSuffix = ".zai.bak"

// Write atomically replaces path's contents with data, creating missing
// parent directories (0700). A symlinked path is resolved first so the link
// itself survives (dotfile managers keep configs as symlinks).
func Write(path string, data []byte, perm os.FileMode) error {
	target, err := resolve(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(fmt.Errorf("failed to write %s: %w", path, err))
	}
	if err := tmp.Chmod(perm); err != nil {
		return fail(fmt.Errorf("failed to set permissions on %s: %w", path, err))
	}
	if err := tmp.Close(); err != nil {
		return fail(fmt.Errorf("failed to close temp file: %w", err))
	}
	if err := os.Rename(tmpPath, target); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save %s: %w", path, err)
	}
	return nil
}

// WriteWithBackup is Write for files another program owns: before the first
// overwrite it copies the existing file to path+BackupSuffix. An existing
// backup is never replaced, so it keeps the state from before go-z-ai first
// touched the file.
func WriteWithBackup(path string, data []byte, perm os.FileMode) error {
	target, err := resolve(path)
	if err != nil {
		return err
	}
	backup := target + BackupSuffix
	if _, err := os.Stat(backup); errors.Is(err, fs.ErrNotExist) {
		if existing, rerr := os.ReadFile(target); rerr == nil {
			if werr := os.WriteFile(backup, existing, perm); werr != nil {
				return fmt.Errorf("failed to back up %s: %w", path, werr)
			}
		}
	}
	return Write(target, data, perm)
}

// resolve follows symlinks in path; a path that does not exist yet is
// returned unchanged.
func resolve(path string) (string, error) {
	target, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		return target, nil
	case errors.Is(err, fs.ErrNotExist):
		return path, nil
	default:
		return "", fmt.Errorf("failed to resolve %s: %w", path, err)
	}
}
