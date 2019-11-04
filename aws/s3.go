package aws

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Settings struct {
	BucketName string
	Prefix     string
	KMSKeyID   string

	ThresholdSize int
}

func (cfg *S3Settings) Merge(otherCfg S3Settings) {
	if cfg.BucketName == "" {
		cfg.BucketName = otherCfg.BucketName
	}

	if cfg.Prefix == "" {
		cfg.Prefix = otherCfg.Prefix
	}

	if cfg.KMSKeyID == "" {
		cfg.KMSKeyID = otherCfg.KMSKeyID
	}

	if cfg.ThresholdSize == 0 {
		cfg.ThresholdSize = otherCfg.ThresholdSize
	}
}

func NewS3Uploader(mgr S3UploadManager, cfg S3Settings) *S3Uploader {
	return &S3Uploader{cfg: cfg, mgr: mgr}
}

type S3Uploader struct {
	cfg S3Settings
	mgr S3UploadManager
}

func (s S3Uploader) Upload(body string) (string, error) {
	maxSize := 51200

	if s.cfg.ThresholdSize != 0 {
		maxSize = s.cfg.ThresholdSize
	}

	if len(body) < maxSize {
		return "", nil // no need to do upload, the size is not over threshold
	}

	if s.cfg.BucketName == "" {
		return "", errors.New("can't upload artifact to s3. Bucket name is not configured")
	}

	prefix := s.cfg.Prefix
	if prefix == "" {
		prefix = "stack-assembly"
	}

	r, err := s.mgr.Upload(&s3manager.UploadInput{
		Bucket:      nilString(s.cfg.BucketName),
		Key:         nilString(prefix),
		SSEKMSKeyId: nilString(s.cfg.KMSKeyID),
		Body:        strings.NewReader(body),
	})
	if err != nil {
		return "", err
	}

	return r.Location, nil
}

type S3UploadManager interface {
	Upload(*s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}
