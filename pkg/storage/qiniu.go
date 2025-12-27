package stores

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/utils"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/sirupsen/logrus"
)

// QiNiuStore represents Qiniu Cloud Storage
type QiNiuStore struct {
	AccessKey  string `env:"QINIU_ACCESS_KEY"`
	SecretKey  string `env:"QINIU_SECRET_KEY"`
	BucketName string `env:"QINIU_BUCKET"`
	// Domain is the bound access domain, e.g., https://static.example.com or http://xxx.bkt.clouddn.com
	Domain string `env:"QINIU_DOMAIN"`
	// Private indicates if the bucket is private (private buckets require signed URLs for download)
	Private bool `env:"QINIU_PRIVATE"`
	// Region is the optional region identifier (auto-detected if empty)
	Region string `env:"QINIU_REGION"`
}

func NewQiNiuStore() Store {
	private := strings.EqualFold(utils.GetEnv("QINIU_PRIVATE"), "true")
	return &QiNiuStore{
		AccessKey:  utils.GetEnv("QINIU_ACCESS_KEY"),
		SecretKey:  utils.GetEnv("QINIU_SECRET_KEY"),
		BucketName: utils.GetEnv("QINIU_BUCKET"),
		Domain:     utils.GetEnv("QINIU_DOMAIN"),
		Private:    private,
		Region:     utils.GetEnv("QINIU_REGION"),
	}
}

func (q *QiNiuStore) getMac() *qbox.Mac {
	return qbox.NewMac(q.AccessKey, q.SecretKey)
}

// makeConfig generates storage.Config, auto-detects region; can still work if detection fails (SDK will auto-discover from UC on first request)
func (q *QiNiuStore) makeConfig() storage.Config {
	useHTTPS := strings.HasPrefix(strings.ToLower(q.Domain), "https://")
	cfg := storage.Config{
		UseHTTPS: useHTTPS,
	}
	// Auto-detect region
	if zone, err := storage.GetRegion(q.AccessKey, q.BucketName); err == nil && zone != nil {
		cfg.Region = zone
	}
	// If you need to force a region, set cfg.Region = &storage.RegionHuadong etc. based on q.Region
	return cfg
}

func (q *QiNiuStore) uploadToken() string {
	p := storage.PutPolicy{
		Scope:   q.BucketName,
		Expires: 3600, // 1小时
	}
	return p.UploadToken(q.getMac())
}

// Write uploads a file using form upload (reads r into memory to get content length, suitable for small/medium files; use multipart upload for large files)
func (q *QiNiuStore) Write(key string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	cfg := q.makeConfig()
	uploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	extra := storage.PutExtra{}
	token := q.uploadToken()

	// Use context.Background() as context
	ctx := context.Background()
	return uploader.Put(ctx, &ret, token, key, bytes.NewReader(data), int64(len(data)), &extra)
}

// Exists checks if a file exists by Stat (612 means not found)
func (q *QiNiuStore) Exists(key string) (bool, error) {
	cfg := q.makeConfig()
	bm := storage.NewBucketManager(q.getMac(), &cfg)
	_, err := bm.Stat(q.BucketName, key)
	if err == nil {
		return true, nil
	}
	if e, ok := err.(*storage.ErrorInfo); ok && e.Code == 612 {
		return false, nil
	}
	return false, err
}

// Delete deletes a file directly
func (q *QiNiuStore) Delete(key string) error {
	cfg := q.makeConfig()
	bm := storage.NewBucketManager(q.getMac(), &cfg)
	return bm.Delete(q.BucketName, key)
}

// Read reads a file via PublicURL (public or signed private) using HTTP GET
func (q *QiNiuStore) Read(key string) (io.ReadCloser, int64, error) {
	u := q.PublicURL(key)
	if u == "" {
		return nil, 0, ErrInvalidPath
	}
	resp, err := http.Get(u)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, 0, &utils.Error{Code: resp.StatusCode, Message: "qiniu read failed"}
	}
	var n int64 = -1
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
			n = v
		}
	}
	return resp.Body, n, nil
}

// PublicURL returns public URL for public buckets; returns signed URL with expiration (default 1 hour) for private buckets
func (q *QiNiuStore) PublicURL(key string) string {
	if q.Domain == "" {
		return ""
	}
	d := q.Domain
	if !strings.HasPrefix(d, "http://") && !strings.HasPrefix(d, "https://") {
		d = "http://" + d
	}
	// Public URL
	pub := storage.MakePublicURLv2(d, key)

	if !q.Private {
		return pub
	}
	// Private download URL (signed, valid for 1 hour)
	deadline := time.Now().Add(1 * time.Hour).Unix()
	return storage.MakePrivateURL(q.getMac(), d, key, deadline)
}

// UpdateCallLogDetails updates call log details (using unified storage)
func UpdateCallLogDetails() {
	logrus.WithFields(logrus.Fields{
		"storage": DefaultStoreKind,
	}).Info("call log details updated")
}

// UploadAudio uploads an audio file (using unified storage)
func UploadAudio(filePath, key string) error {
	store := Default()

	// Read local file
	file, err := os.Open(filePath)
	if err != nil {
		logrus.WithError(err).Error("failed to open audio file")
		return err
	}
	defer file.Close()

	// Upload to unified storage
	err = store.Write(key, file)
	if err != nil {
		logrus.WithError(err).Error("failed to upload audio")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"filePath": filePath,
		"key":      key,
		"storage":  DefaultStoreKind,
	}).Info("audio uploaded successfully")

	return nil
}

// UploadTrace uploads a trace file (using unified storage)
func UploadTrace(filePath, key string) error {
	store := Default()

	// Read local file
	file, err := os.Open(filePath)
	if err != nil {
		logrus.WithError(err).Error("failed to open trace file")
		return err
	}
	defer file.Close()

	// Upload to unified storage
	err = store.Write(key, file)
	if err != nil {
		logrus.WithError(err).Error("failed to upload trace")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"filePath": filePath,
		"key":      key,
		"storage":  DefaultStoreKind,
	}).Info("trace uploaded successfully")

	return nil
}

// The following functions are kept for backward compatibility but are deprecated, please use the unified functions above
// Deprecated: Use UploadAudio instead
func UploadAudioToQiniu(filePath, key string) error {
	return UploadAudio(filePath, key)
}

// Deprecated: Use UploadTrace instead
func UploadTraceToQiniu(filePath, key string) error {
	return UploadTrace(filePath, key)
}

// Deprecated: Use UpdateCallLogDetails instead
func UpdateCallLogDetailsQiniu() {
	UpdateCallLogDetails()
}
