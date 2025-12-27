package stores

import (
	"io"
	"net/http"

	"github.com/code-100-precent/LingFramework/pkg/utils"
)

const (
	KindLocal = "local" // Local file system storage
	KindCos   = "cos"   // Tencent Cloud Object Storage
	KindMinio = "minio" // MinIO / S3 compatible storage
	KindQiNiu = "qiniu" // Qiniu Cloud Storage
	KindOSS   = "oss"   // Alibaba Cloud Object Storage Service
	KindS3    = "s3"    // Amazon S3
)

var ErrInvalidPath = &utils.Error{Code: http.StatusBadRequest, Message: "invalid path"}

// DefaultStoreKind is the default storage type, read from STORAGE_KIND environment variable
// Valid values: local, qiniu, cos, minio, oss, s3
// Defaults to local if not set
var DefaultStoreKind = getDefaultStoreKind()

// getDefaultStoreKind gets default storage type from environment variable
func getDefaultStoreKind() string {
	kind := utils.GetEnv("STORAGE_KIND")
	if kind == "" {
		return KindLocal
	}
	// Validate storage type
	switch kind {
	case KindLocal, KindCos, KindMinio, KindQiNiu, KindOSS, KindS3:
		return kind
	default:
		// Invalid type, use default
		return KindLocal
	}
}

// Store is the common storage interface
type Store interface {
	// Read reads a file from storage
	Read(key string) (io.ReadCloser, int64, error)
	// Write writes a file to storage
	Write(key string, r io.Reader) error
	// Delete deletes a file from storage
	Delete(key string) error
	// Exists checks if a file exists in storage
	Exists(key string) (bool, error)
	// PublicURL returns the public URL for a file
	PublicURL(key string) string
}

// GetStore creates a storage instance by kind
func GetStore(kind string) Store {
	switch kind {
	case KindCos:
		return NewCosStore()
	case KindMinio:
		return NewMinioStore()
	case KindQiNiu:
		return NewQiNiuStore()
	case KindOSS:
		return NewOSSStore()
	case KindS3:
		return NewS3Store()
	default:
		return NewLocalStore()
	}
}

// Default returns the default storage instance
func Default() Store {
	return GetStore(DefaultStoreKind)
}
