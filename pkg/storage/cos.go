package stores

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/code-100-precent/LingFramework/pkg/utils"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// CosStore represents Tencent Cloud Object Storage
type CosStore struct {
	SecretID   string `env:"COS_SECRET_ID"`
	SecretKey  string `env:"COS_SECRET_KEY"`
	Region     string `env:"COS_REGION"`
	BucketName string `env:"COS_BUCKET_NAME"`
	Domain     string `env:"COS_DOMAIN"` // Custom domain for public access
}

// NewCosStore creates a new COS storage instance
func NewCosStore() Store {
	return &CosStore{
		SecretID:   utils.GetEnv("COS_SECRET_ID"),
		SecretKey:  utils.GetEnv("COS_SECRET_KEY"),
		Region:     utils.GetEnv("COS_REGION"),
		BucketName: utils.GetEnv("COS_BUCKET_NAME"),
		Domain:     utils.GetEnv("COS_DOMAIN"),
	}
}

// client creates and returns a COS client
func (c *CosStore) client() (*cos.Client, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", c.BucketName, c.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to parse COS URL: %w", err)
	}
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  c.SecretID,
			SecretKey: c.SecretKey,
		},
	})
	return client, nil
}

// Delete deletes a file from COS
func (c *CosStore) Delete(key string) error {
	client, err := c.client()
	if err != nil {
		return err
	}
	_, err = client.Object.Delete(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// Exists checks if a file exists in COS
func (c *CosStore) Exists(key string) (bool, error) {
	client, err := c.client()
	if err != nil {
		return false, err
	}
	ok, err := client.Object.IsExist(context.Background(), key)
	if err != nil {
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}
	return ok, nil
}

// Read reads a file from COS
func (c *CosStore) Read(key string) (io.ReadCloser, int64, error) {
	client, err := c.client()
	if err != nil {
		return nil, 0, err
	}

	// Get object meta to get size
	head, err := client.Object.Head(context.Background(), key, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object head: %w", err)
	}
	defer head.Body.Close()

	var size int64
	if head.ContentLength > 0 {
		size = head.ContentLength
	}

	// Get object
	resp, err := client.Object.Get(context.Background(), key, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object: %w", err)
	}

	return resp.Body, size, nil
}

// Write writes a file to COS
func (c *CosStore) Write(key string, r io.Reader) error {
	client, err := c.client()
	if err != nil {
		return err
	}
	_, err = client.Object.Put(context.Background(), key, r, nil)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

// PublicURL returns the public URL for a file
func (c *CosStore) PublicURL(key string) string {
	// If custom domain is set, use it
	if c.Domain != "" {
		domain := strings.TrimSuffix(c.Domain, "/")
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
		return fmt.Sprintf("%s/%s", domain, strings.TrimPrefix(key, "/"))
	}

	// Otherwise, use COS URL format
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", c.BucketName, c.Region, strings.TrimPrefix(key, "/"))
}
