package compression

import (
	"audio_compression/config"
	"audio_compression/entity"
	"audio_compression/internal/storage/s3repo"
	"audio_compression/pkg/archive"
	"audio_compression/pkg/logger"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"gorm.io/gorm"
)

type CompressionUsecase struct {
	StorageRepo           entity.StorageRepository
	uncompressedArchiever archive.Archiver
	compressedArchiever   archive.Archiver
	CompressionRepo       *CompressionRepository
	l                     logger.Interface
	compBuffer            CompressionBuffer
	decompBuffer          CompressionBuffer
}

type CompressionBuffer struct {
	sourceBuffer *bytes.Buffer
	outputBuffer *bytes.Buffer
}

func NewCompressionUsecase(cfg *config.Config, db *gorm.DB, l logger.Interface) *CompressionUsecase {
	s3Repo, err := s3repo.NewS3Repository()
	if err != nil {
		l.Error(err)
		l.Fatal("Failed to init S3 Repository")
	}
	uncompArchiever := archive.NewTarArchiever()
	compArchiever := archive.NewTarGzArchiever()

	compRepo := NewCompressionRepository(db, l)

	compBuffer := CompressionBuffer{new(bytes.Buffer), new(bytes.Buffer)}
	decompBuffer := CompressionBuffer{new(bytes.Buffer), new(bytes.Buffer)}

	cu := &CompressionUsecase{s3Repo, uncompArchiever, compArchiever, compRepo, l, compBuffer, decompBuffer}

	return cu
}

func (c *CompressionUsecase) PlanCompression(ctx context.Context, bucket, key string) error {
	return nil
}

func (c *CompressionUsecase) GetDecompression(ctx context.Context, bucket, key string) ([]byte, error) {
	ctx, span := otel.Tracer(traceName).Start(ctx, "GetDecompression")
	defer span.End()

	span.SetAttributes(attribute.String("bucket", bucket))
	span.SetAttributes(attribute.String("key", key))

	if !c.CompressionRepo.IsDecompressHasRequested(ctx, bucket, key) {
		span.AddEvent("Starting DoDecompression")

		c.CompressionRepo.CreateDecompression(ctx, bucket, key)

		go func() {
			filepath, err := c.DoDecompression(ctx, bucket, key)
			if err != nil {
				c.CompressionRepo.UpdateDecompressionError(ctx, bucket, key, err)
			}
			if filepath != "" {
				c.CompressionRepo.UpdateDecompressionFilePath(ctx, bucket, key, filepath)
			}
		}()
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*80)
	defer cancel()

	fileContentChan := make(chan []byte)
	errorChan := make(chan error)

	go func() {
		for {
			filepath, err := c.CompressionRepo.GetDecompressedObjectResult(ctx, bucket, key)
			if filepath != "" {
				span.AddEvent("Found decompressed object file path")
				content, err := ioutil.ReadFile(filepath)
				if err != nil {
					c.l.Error(err)
				}
				fileContentChan <- content
				return
			}
			if err != nil {
				span.AddEvent("Found decompressed object error")
				errorChan <- err
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {
	case <-ctxTimeout.Done():
		return nil, ctxTimeout.Err()
	case fileContent := <-fileContentChan:
		return fileContent, nil
	case err := <-errorChan:
		return nil, err
	}
}

func (c *CompressionUsecase) DoCompression(ctx context.Context, bucket, key string) (error, bool) {
	ctx, span := otel.Tracer(traceName).Start(ctx, "DoCompression")
	defer span.End()

	span.SetAttributes(attribute.String("bucket", bucket))
	span.SetAttributes(attribute.String("key", key))

	if !c.isKeyExtensionValid(key, ".tar") {
		return errors.New("Invalid file extension"), false
	}

	sourceBuffer := c.compBuffer.sourceBuffer
	outputBuffer := c.compBuffer.outputBuffer

	sourceBuffer.Reset()
	outputBuffer.Reset()

	// Download from s3
	if err := c.StorageRepo.DownloadObject(ctx, bucket, key, sourceBuffer); err != nil {
		var responseError *awshttp.ResponseError
		if errors.As(err, &responseError) && responseError.ResponseError.HTTPStatusCode() == http.StatusNotFound {
			return err, false
		}
		return err, true
	}

	// Extract
	files, err := c.uncompressedArchiever.Extract(ctx, sourceBuffer)
	if err != nil {
		return err, false
	}

	var newFiles []entity.FileObject

	// Convert wav to flac
	for _, file := range files {
		newFiles = append(newFiles, file)
	}

	// Compress to tar gz
	if err := c.compressedArchiever.Compress(ctx, newFiles, outputBuffer); err != nil {
		return err, true
	}

	compressedBucket := bucket + "-compressed"
	compressedKey := key + ".gz"

	// Upload to S3
	if err := c.StorageRepo.UploadObject(ctx, compressedBucket, compressedKey, outputBuffer); err != nil {
		return err, true
	}

	return nil, false
}

func (c *CompressionUsecase) DoDecompression(ctx context.Context, bucket, key string) (string, error) {
	ctx, span := otel.Tracer(traceName).Start(ctx, "DoDecompression")
	defer span.End()

	span.SetAttributes(attribute.String("bucket", bucket))
	span.SetAttributes(attribute.String("key", key))

	if !c.isKeyExtensionValid(key, ".tar") {
		return "", errors.New("Invalid file extension")
	}

	compressedBucket := bucket + "-compressed"
	compressedKey := key + ".gz"

	fmt.Print(compressedKey)

	// sourceBuffer := c.decompBuffer.sourceBuffer
	// outputBuffer := c.decompBuffer.outputBuffer

	// sourceBuffer.Reset()
	// outputBuffer.Reset()

	sourceBuffer := new(bytes.Buffer)
	outputBuffer := new(bytes.Buffer)

	// Download from s3
	c.l.Debug("Downloading object from S3")
	if err := c.StorageRepo.DownloadObject(ctx, compressedBucket, compressedKey, sourceBuffer); err != nil {
		return "", err
	}

	// Extract
	c.l.Debug("Extracting object")
	files, err := c.compressedArchiever.Extract(ctx, sourceBuffer)
	if err != nil {
		return "", err
	}

	var newFiles []entity.FileObject
	// ctx := context.TODO()

	// Convert wav to flac
	c.l.Debug("Walk the files...")
	for _, file := range files {
		newFiles = append(newFiles, file)
	}

	// Compress to tar gz
	if err := c.uncompressedArchiever.Compress(ctx, newFiles, outputBuffer); err != nil {
		return "", err
	}

	// Put decompression result to tempfile
	filePath, err := c.PutToTempFile(ctx, outputBuffer)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func (c *CompressionUsecase) isKeyExtensionValid(key, ext string) bool {
	fileExtension := filepath.Ext(key)
	if fileExtension != ext {
		return false
	}
	return true
}

func (cr *CompressionUsecase) PutToTempFile(ctx context.Context, r io.Reader) (string, error) {
	ctx, span := otel.Tracer(traceName).Start(ctx, "PutToTempFile")
	defer span.End()

	// Put to tempfile
	f, err := os.CreateTemp(".", "decompressed_audio_*.tar")
	if err != nil {
		return "", err
	}
	defer f.Close()

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	filePath := f.Name()

	if _, err := f.Write(body); err != nil {
		return "", err
	}

	return filePath, nil
}
