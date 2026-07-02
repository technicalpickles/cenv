package env

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/technicalpickles/cenv/internal/auth"
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-_]*$`)

// BasePath returns the base directory for cenv environments.
// It uses CENV_BASE if set, otherwise defaults to ~/.local/share/cenv.
func BasePath() string {
	if base := os.Getenv("CENV_BASE"); base != "" {
		return base
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to a reasonable default if home dir lookup fails
		return filepath.Join(".local", "share", "cenv")
	}
	return filepath.Join(home, ".local", "share", "cenv")
}

// Path returns the full path for a named environment.
func Path(name string) string {
	return filepath.Join(BasePath(), name)
}

// Exists reports whether the named environment directory exists.
func Exists(name string) bool {
	info, err := os.Stat(Path(name))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// List returns the names of all environment directories in BasePath.
// Files are ignored; only directories are returned.
func List() ([]string, error) {
	base := BasePath()
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading base path %q: %w", base, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	if names == nil {
		names = []string{}
	}
	return names, nil
}

// Remove deletes the named environment directory.
// It returns an error if the environment does not exist.
func Remove(name string) error {
	p := Path(name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf("environment %q not found", name)
	}
	return os.RemoveAll(p)
}

// Info describes an environment's on-disk metadata.
type Info struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	HasAuth bool      `json:"has_auth"`
	Size    int64     `json:"size"`
	Mtime   time.Time `json:"mtime"`
}

// Inspect returns metadata for the named environment. Size is the sum of all
// file sizes under the env directory. Mtime is the most recent modification
// time of any file in the env, or the directory's own mtime if empty.
// HasAuth reports whether the env has a detectable auth config.
func Inspect(name string) (*Info, error) {
	p := Path(name)
	stat, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("env %q is not a directory", name)
	}

	var size int64
	mtime := stat.ModTime()
	err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		size += fi.Size()
		if m := fi.ModTime(); m.After(mtime) {
			mtime = m
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking env %q: %w", name, err)
	}

	authErr := auth.Detect(p)
	return &Info{
		Name:    name,
		Path:    p,
		HasAuth: authErr == nil,
		Size:    size,
		Mtime:   mtime,
	}, nil
}

// ValidateName checks that name is a valid environment name.
// Names must start with a letter and may contain letters, numbers, hyphens, and underscores.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if !validNameRe.MatchString(name) {
		return fmt.Errorf("invalid name %q: must start with a letter and contain only letters, numbers, hyphens, and underscores", name)
	}
	return nil
}
