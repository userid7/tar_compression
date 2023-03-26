package compression

import (
	"audio_compression/entity"
	"audio_compression/pkg/logger"
	"context"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"
)

type CompressionRepository struct {
	mu *sync.Mutex
	db *gorm.DB
	l  logger.Interface

	dos []entity.DecompressionObject
}

func NewCompressionRepository(db *gorm.DB, l logger.Interface) *CompressionRepository {
	var mut sync.Mutex
	var dos []entity.DecompressionObject
	repo := &CompressionRepository{&mut, db, l, dos}
	go repo.decompressionListWorker()
	return repo
}

func (cr *CompressionRepository) IsCompressed(ctx context.Context, bucket, key string) bool {
	return false
}

func (cr *CompressionRepository) CreateCompression(ctx context.Context, bucket, key string) {
	// cr.db.Create()
}

func (cr *CompressionRepository) CreateDecompression(ctx context.Context, bucket, key string) {
	obj := entity.DecompressionObject{Bucket: bucket, Key: key, LastAccess: time.Now(), TTL: time.Minute}
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.dos = append(cr.dos, obj)
}
func (cr *CompressionRepository) UpdateDecompressionFilePath(ctx context.Context, bucket, key, filepath string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	for i, obj := range cr.dos {
		if obj.Bucket == bucket && obj.Key == key {
			cr.dos[i].FilePath = filepath
		}
	}
}

func (cr *CompressionRepository) UpdateDecompressionError(ctx context.Context, bucket, key string, err error) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	for i, obj := range cr.dos {
		if obj.Bucket == bucket && obj.Key == key {
			cr.dos[i].Err = err
		}
	}
}

func (cr *CompressionRepository) decompressionListWorker() {
	for {
		// fmt.Printf("%v \n", cr.dos)
		for i, obj := range cr.dos {
			if time.Now().Sub(obj.LastAccess) > obj.TTL {
				// Removing object from list if already expired
				cr.l.Info("Removing decompression object from list : %s - %s", obj.Bucket, obj.Key)
				cr.mu.Lock()
				os.Remove(obj.FilePath)
				cr.dos = append(cr.dos[:i], cr.dos[i+1:]...)
				cr.mu.Unlock()
				break
			}
		}
		// debug.FreeOSMemory()
		time.Sleep(1 * time.Second)
	}
}

func (cr *CompressionRepository) IsDecompressHasRequested(ctx context.Context, bucket, key string) bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	for _, obj := range cr.dos {
		if obj.Bucket == bucket && obj.Key == key {
			return true
		}
	}

	return false
}

func (cr *CompressionRepository) GetDecompressedObjectResult(ctx context.Context, bucket, key string) (string, error) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	for _, obj := range cr.dos {
		if obj.Bucket == bucket && obj.Key == key {
			return obj.FilePath, obj.Err
		}
	}

	return "", nil
}
