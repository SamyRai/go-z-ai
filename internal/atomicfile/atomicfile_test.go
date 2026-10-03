package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCreatesParentsAndPerms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "f.json")
	if err := Write(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("Write: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("stat: %v mode %v", err, info.Mode())
	}
}

// The backup keeps the pre-first-write contents across repeated writes.
func TestWriteWithBackupKeepsOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"first", "second"} {
		if err := WriteWithBackup(path, []byte(v), 0o600); err != nil {
			t.Fatalf("WriteWithBackup: %v", err)
		}
	}
	if b, _ := os.ReadFile(path + BackupSuffix); string(b) != "original" {
		t.Errorf("backup = %q, want the original contents", b)
	}
	if b, _ := os.ReadFile(path); string(b) != "second" {
		t.Errorf("file = %q, want second", b)
	}
}

// Writing through a symlink updates the target and keeps the link.
func TestWriteFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "link.json")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unsupported:", err)
	}
	if err := WriteWithBackup(link, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteWithBackup: %v", err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link was replaced: %v %v", fi, err)
	}
	if b, _ := os.ReadFile(target); string(b) != "new" {
		t.Errorf("target = %q, want new", b)
	}
}
