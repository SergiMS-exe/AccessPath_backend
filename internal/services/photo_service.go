package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"time"

	"github.com/chai2010/webp"
	"github.com/minio/minio-go/v7"
)

// PhotoService handles image validation, WebP conversion and MinIO upload.
// It does NOT write to the database; callers must persist the returned URL
// via PhotoRepository.SaveTx inside their own transaction.
type PhotoService struct {
	minio  *minio.Client
	bucket string
}

func NewPhotoService(minioClient *minio.Client, bucket string) *PhotoService {
	return &PhotoService{minio: minioClient, bucket: bucket}
}

// Upload validates raw image bytes, encodes them as WebP and uploads to MinIO.
// Returns the public object URL on success.
func (s *PhotoService) Upload(ctx context.Context, data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("photo: unsupported or invalid image: %w", err)
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Quality: 80}); err != nil {
		return "", fmt.Errorf("photo: webp encode: %w", err)
	}

	objectName := fmt.Sprintf("reviews/%d.webp", time.Now().UnixNano())
	_, err = s.minio.PutObject(ctx, s.bucket, objectName, &buf, int64(buf.Len()),
		minio.PutObjectOptions{ContentType: "image/webp"})
	if err != nil {
		return "", fmt.Errorf("photo: minio upload: %w", err)
	}

	scheme := "http"
	if s.minio.IsSecure() {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/%s/%s", scheme, s.minio.EndpointURL().Host, s.bucket, objectName)
	return url, nil
}
