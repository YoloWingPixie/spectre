package audit

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed assets schema templates SKILL.md docs
var resources embed.FS

func resourceDir() (string, error) {
	hash := sha256.New()
	err := fs.WalkDir(resources, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := resources.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(hash, "%d:%s:%d:", len(path), path, len(data))
		hash.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	parent := filepath.Join(cache, "spectre", "audit")
	destination := filepath.Join(parent, fmt.Sprintf("%x", hash.Sum(nil)))
	if info, err := os.Stat(destination); err == nil && info.IsDir() {
		return destination, nil
	}
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", err
	}
	temporary, err := os.MkdirTemp(parent, ".extract-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temporary)
	err = fs.WalkDir(resources, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(temporary, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := resources.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
	if err != nil {
		return "", err
	}
	if err := os.Rename(temporary, destination); err != nil {
		if info, statErr := os.Stat(destination); statErr != nil || !info.IsDir() {
			return "", err
		}
	}
	return destination, nil
}
