package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"

	"audio_compression/entity"

	"go.opentelemetry.io/otel"
)

type TarGzArchiever struct {
}

func NewTarGzArchiever() Archiver {
	return &TarGzArchiever{}
}

func (gz *TarGzArchiever) Compress(ctx context.Context, fileObjects []entity.FileObject, buf io.Writer) error {
	ctx, span := otel.Tracer(traceName).Start(ctx, "compress - tar gz")
	defer span.End()

	gw := gzip.NewWriter(buf)
	defer gw.Close()

	tw := tar.NewWriter(gw)
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

func (gz *TarGzArchiever) Extract(ctx context.Context, buf io.Reader) ([]entity.FileObject, error) {
	ctx, span := otel.Tracer(traceName).Start(ctx, "extract - tar gz")
	defer span.End()

	var extractedFiles []entity.FileObject

	gr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(gr)
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
