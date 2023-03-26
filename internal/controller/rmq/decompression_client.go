package rmq

import (
	"audio_compression/config"
	"audio_compression/entity"
	"audio_compression/internal/compression"
	"audio_compression/pkg/logger"
	"context"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type DecompressionClient struct {
	l               logger.Interface
	blobStorageRepo entity.StorageRepository
	cu              *compression.CompressionUsecase
	reqMap          map[string]entity.CompressionResponse
	mu              sync.Mutex
}

func NewDecompressionClient(cfg *config.Config, l logger.Interface) *DecompressionClient {
	rand.Seed(time.Now().UnixNano())
	return &DecompressionClient{l: l, reqMap: make(map[string]entity.CompressionResponse)}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (dc *DecompressionClient) GetOrCreateRequest(bucket, key, compType string) (string, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	for key, val := range dc.reqMap {
		if val.Bucket == bucket && val.Key == key && val.Type == compType {
			return key, true
		}
	}

	corrId := randSeq(10)

	dc.reqMap[corrId] = entity.CompressionResponse{Bucket: bucket, Key: key, Type: compType}

	return corrId, false
}

func (dc *DecompressionClient) SetDecompressionResponse(corrId string, res entity.CompressionResponse) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	// bs, _ := json.Marshal(dc.reqMap)
	// fmt.Println(string(bs))
	if _, ok := dc.reqMap[corrId]; ok {
		dc.reqMap[corrId] = res
	}
}

func (dc *DecompressionClient) GetDecompressionResponse(ctx context.Context, corrId string, timeOut int) (entity.CompressionResponse, error) {
	resultChan := make(chan entity.CompressionResponse)

	// ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*100)
	defer cancel()

	go func() {
		for {
			dc.mu.Lock()
			val, ok := dc.reqMap[corrId]
			dc.mu.Unlock()
			if ok {
				if val.ResultAddress != "" {
					resultChan <- val
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {
	case res := <-resultChan:
		if res.Err != nil {
			return entity.CompressionResponse{}, res.Err
		}
		return res, nil
	case <-ctx.Done():
		return entity.CompressionResponse{}, errors.New("response timeout exceed")
	}
}

// TODO : refactor to use blobStorage
func (dc *DecompressionClient) GetByteFromFileSystem(address string) ([]byte, error) {
	return os.ReadFile(address)
}
