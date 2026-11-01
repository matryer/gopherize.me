package server

import (
	"github.com/pkg/errors"
	"os"
)

// DefaultBucketName returns the existing artwork and saved-gopher bucket.
func DefaultBucketName() (string, error) {
	bucket := os.Getenv("BUCKET_NAME")
	if bucket == "" {
		return "", errors.New("BUCKET_NAME must be configured")
	}
	return bucket, nil
}
