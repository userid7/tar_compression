package entity

import (
	"context"
	"time"
)

type CompressionUsecase interface {
	PlanCompression(ctx context.Context, bucket, key string) error
	GetDecompression(ctx context.Context, bucket, key string) ([]byte, error)
}

type CompressionRequest struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	Type   string
}

type CompressionResponse struct {
	Bucket        string `json:"bucket"`
	Key           string `json:"key"`
	Type          string
	ResultType    string
	ResultAddress string
	Err           error
}

type DecompressionObject struct {
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	FilePath   string
	Err        error
	LastAccess time.Time
	TTL        time.Duration
}
