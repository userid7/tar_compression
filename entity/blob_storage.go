package entity

import (
	"context"
	"io"
)

type StorageRepository interface {
	DownloadObject(ctx context.Context, bucket string, key string, w io.Writer) error
	UploadObject(ctx context.Context, bucket string, key string, r io.Reader) error
}
