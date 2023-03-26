package archive

import (
	"audio_compression/entity"
	"context"
	"io"
)

type Archiver interface {
	Compress(ctx context.Context, fileObjects []entity.FileObject, buf io.Writer) error
	Extract(ctx context.Context, r io.Reader) ([]entity.FileObject, error)
}
