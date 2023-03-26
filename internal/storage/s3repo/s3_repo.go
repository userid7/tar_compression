package s3repo

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.opentelemetry.io/otel"
)

const traceName = "S3-Repo"

type S3Repository struct {
	sess *s3.Client
}

func boolPointer(b bool) *bool {
	return &b
}

func NewS3Repository() (*S3Repository, error) {
	// sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	// if err != nil {
	// 	fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
	// 	fmt.Println(err)
	// 	return nil, err
	// }

	const defaultRegion = "us-east-1"
	hostAddress := "http://localhost:9000"

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:       "aws",
			SigningRegion:     defaultRegion,
			URL:               hostAddress,
			HostnameImmutable: true,
		}, nil
	})

	cfg := aws.Config{
		Region:                      defaultRegion,
		EndpointResolverWithOptions: resolver,
		Credentials:                 credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""),
	}

	s3Client := s3.NewFromConfig(cfg)
	return &S3Repository{s3Client}, nil
}

func (s3Repo *S3Repository) DownloadObject(ctx context.Context, bucket string, key string, w io.Writer) error {
	ctx, span := otel.Tracer(traceName).Start(ctx, "DownloadObject")
	defer span.End()

	downloader := manager.NewDownloader(s3Repo.sess)

	var buffer []byte
	bw := manager.NewWriteAtBuffer(buffer)

	numBytes, err := downloader.Download(context.TODO(), bw, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	if numBytes < 1 {
		return errors.New("zero bytes written to memory")
	}

	if _, err := w.Write(bw.Bytes()); err != nil {
		return err
	}

	return nil
}

func (s3Repo *S3Repository) UploadObject(ctx context.Context, bucket string, key string, r io.Reader) error {
	ctx, span := otel.Tracer(traceName).Start(ctx, "UploadObject")
	defer span.End()

	uploader := manager.NewUploader(s3Repo.sess)

	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return err
	}

	return nil
}
