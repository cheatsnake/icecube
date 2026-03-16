package imagestore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/pkg/fs"
	"github.com/cheatsnake/icecube/internal/pkg/uuid"
)

type BlobStoreS3 struct {
	client *s3.Client
	bucket string
	prefix string
}

func NewBlobStoreS3(client *s3.Client, bucket, prefix string) *BlobStoreS3 {
	return &BlobStoreS3{client: client, bucket: bucket, prefix: prefix}
}

func (s *BlobStoreS3) UploadImage(ctx context.Context, r io.Reader, name string, size int64) (*image.Variant, error) {
	imageID := uuid.V7()
	key := s.objectKeyByID(imageID)

	// Read entire body into buffer - must be seekable for S3 SDK checksum
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload data: %w", err)
	}

	buffer := bytes.NewReader(data)
	meta, _, err := fs.GetImageMetadataFromReader(buffer)
	if err != nil {
		return nil, err
	}

	// Reset buffer position after reading for metadata
	buffer.Seek(0, io.SeekStart)

	imageFormat := image.Format(meta.Format)
	if err := image.ValidateFormat(imageFormat); err != nil {
		return nil, errors.Join(errs.ErrInvalidInput, fmt.Errorf("invalid image format: %w", err))
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          buffer,
		ContentType:   aws.String(contentTypeForFormat(meta.Format)),
		ContentLength: &size,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 put object: %w", err)
	}

	return image.NewVariant(imageID, sanitizeFilename(name), imageFormat, meta.Width, meta.Height, size)
}

func (s *BlobStoreS3) DownloadImage(ctx context.Context, id string) (io.ReadCloser, error) {
	key := s.objectKeyByID(id)

	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *s3types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, errors.Join(errs.ErrNotFound, errors.New("object not found: "+id))
		}
		return nil, fmt.Errorf("s3 get object: %w", err)
	}

	return out.Body, nil
}

func (s *BlobStoreS3) DeleteImage(ctx context.Context, id string) error {
	key := s.objectKeyByID(id)

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete object: %w", err)
	}
	return nil
}

func (s *BlobStoreS3) DeleteImages(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	const batchSize = 1000
	var aggErrs []string

	for start := 0; start < len(ids); start += batchSize {
		end := min(start+batchSize, len(ids))

		objects := make([]s3types.ObjectIdentifier, 0, end-start)
		for _, id := range ids[start:end] {
			objects = append(objects, s3types.ObjectIdentifier{Key: aws.String(s.objectKeyByID(id))})
		}

		_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &s3types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			aggErrs = append(aggErrs, err.Error())
		}
	}

	if len(aggErrs) > 0 {
		return fmt.Errorf("some deletes failed: %s", strings.Join(aggErrs, "; "))
	}
	return nil
}

func (s *BlobStoreS3) objectKeyByID(id string) string {
	if s.prefix != "" {
		return filepath.ToSlash(filepath.Join(s.prefix, id))
	}

	return filepath.ToSlash(id)
}

func contentTypeForFormat(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
