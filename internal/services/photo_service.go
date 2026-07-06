package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// PhotoService handles image validation, WebP conversion and MinIO upload.
// It does NOT write to the database; callers must persist the returned URL
// via PhotoRepository.SaveTx inside their own transaction.
type PhotoService struct {
	minio         *minio.Client
	bucket        string
	publicBaseURL string // resoluble desde el cliente; vacio => endpoint interno (dev)
}

func NewPhotoService(minioClient *minio.Client, bucket, publicBaseURL string) *PhotoService {
	return &PhotoService{minio: minioClient, bucket: bucket, publicBaseURL: publicBaseURL}
}

// Upload validates raw image bytes, re-encodes them as JPEG and uploads to MinIO.
// Returns the public object URL and the object key on success.
func (s *PhotoService) Upload(ctx context.Context, data []byte) (url, objectKey string, err error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("photo: unsupported or invalid image: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return "", "", fmt.Errorf("photo: jpeg encode: %w", err)
	}

	objectKey = fmt.Sprintf("submissions/%d.jpg", time.Now().UnixNano())
	_, err = s.minio.PutObject(ctx, s.bucket, objectKey, &buf, int64(buf.Len()),
		minio.PutObjectOptions{ContentType: "image/jpeg"})
	if err != nil {
		return "", "", fmt.Errorf("photo: minio upload: %w", err)
	}

	// Servir por URL publica resoluble desde el movil (B2). Si no se configura una
	// base publica, caer al endpoint interno de MinIO (solo desarrollo local).
	if s.publicBaseURL != "" {
		url = fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.publicBaseURL, "/"), s.bucket, objectKey)
	} else {
		endpointURL := s.minio.EndpointURL()
		url = fmt.Sprintf("%s://%s/%s/%s", endpointURL.Scheme, endpointURL.Host, s.bucket, objectKey)
	}
	return url, objectKey, nil
}
