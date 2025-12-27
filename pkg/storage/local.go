package stores

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/code-100-precent/LingFramework/pkg/utils"
)

// UploadDir is the default upload directory for local storage
var UploadDir string = "./uploads"

// MediaPrefix defines the public URL prefix for locally stored files
// Defaults to "/uploads" to align with other upload endpoints
var MediaPrefix string = "/uploads"

// LocalStore represents local file system storage
type LocalStore struct {
	Root       string
	NewDirPerm os.FileMode
}

// Delete deletes a file from local storage
func (l *LocalStore) Delete(key string) error {
	// Ensure Root is an absolute path
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return err
	}

	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root) {
		return ErrInvalidPath
	}
	return os.Remove(fname)
}

// Exists checks if a file exists in local storage
func (l *LocalStore) Exists(key string) (bool, error) {
	// Ensure Root is an absolute path
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return false, err
	}

	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root) {
		return false, ErrInvalidPath
	}
	_, err = os.Stat(fname)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Read reads a file from local storage
func (l *LocalStore) Read(key string) (io.ReadCloser, int64, error) {
	// Ensure Root is an absolute path
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return nil, 0, err
	}

	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root) {
		return nil, 0, ErrInvalidPath
	}
	st, err := os.Stat(fname)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, 0, err
	}
	return f, st.Size(), nil
}

// Write writes a file to local storage
func (l *LocalStore) Write(key string, r io.Reader) error {
	// Ensure Root is an absolute path
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return err
	}

	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root) {
		return ErrInvalidPath
	}
	dir := filepath.Dir(fname)
	err = os.MkdirAll(dir, l.NewDirPerm)
	if err != nil {
		return err
	}
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

// PublicURL returns the public URL for a file in local storage
func (l *LocalStore) PublicURL(key string) string {
	mediaPrefix := utils.GetEnv("MEDIA_PREFIX")
	if mediaPrefix == "" {
		mediaPrefix = MediaPrefix
	}
	// Use path.Join instead of filepath.Join to ensure URL paths always use forward slashes
	// Also normalize the path by removing extra slashes
	mediaPrefix = strings.TrimSuffix(mediaPrefix, "/")
	key = strings.TrimPrefix(key, "/")
	return path.Join("/", mediaPrefix, key)
}

// NewLocalStore creates a new local storage instance
func NewLocalStore() Store {
	uploadDir := utils.GetEnv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = UploadDir
	}
	s := &LocalStore{
		Root:       uploadDir,
		NewDirPerm: 0755,
	}
	return s
}
