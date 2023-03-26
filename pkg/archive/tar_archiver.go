package archive

import (
	"archive/tar"
	"context"
	"io"

	"audio_compression/entity"

	"go.opentelemetry.io/otel"
)

type TarArchiever struct {
}

func NewTarArchiever() Archiver {
	return &TarArchiever{}
}

func (gz *TarArchiever) Compress(ctx context.Context, fileObjects []entity.FileObject, buf io.Writer) error {
	ctx, span := otel.Tracer(traceName).Start(ctx, "compress - tar")
	defer span.End()

	tw := tar.NewWriter(buf)
	defer tw.Close()

	for _, fileObject := range fileObjects {
		hdr := &tar.Header{
			Name: fileObject.Name,
			Mode: int64(0600),
			Size: int64(len(fileObject.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(fileObject.Body); err != nil {
			return err
		}
	}
	return nil
}

func (gz *TarArchiever) Extract(ctx context.Context, buf io.Reader) ([]entity.FileObject, error) {
	ctx, span := otel.Tracer(traceName).Start(ctx, "extract - tar")
	defer span.End()

	var extractedFiles []entity.FileObject
	tr := tar.NewReader(buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		fileHeader := hdr

		fileBody, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}

		extractedFiles = append(extractedFiles, entity.FileObject{Name: fileHeader.Name, Body: fileBody})

	}
	return extractedFiles, nil
}
