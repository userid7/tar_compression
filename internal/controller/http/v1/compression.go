package v1

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"

	"audio_compression/entity"
	"audio_compression/pkg/logger"
	// "github.com/evrone/go-clean-template/internal/entity"
	// "github.com/evrone/go-clean-template/internal/usecase"
	// "github.com/evrone/go-clean-template/pkg/logger"
)

type compressionRoutes struct {
	cu entity.CompressionUsecase
	l  logger.Interface
}

func newCompressionRoutes(handler *gin.RouterGroup, cu entity.CompressionUsecase, l logger.Interface) {
	r := &compressionRoutes{cu, l}

	h := handler.Group("/compression")
	{
		h.GET("/compress/:bucket/*key", r.compress)
		h.GET("/decompress/:bucket/*key", r.decompress)
	}
}

// @Summary     trigger compression
// @Description trigger compression using webhook
// @ID          compression
// @Tags  	    compress
// @Produce     json
// @Success     200
// @Failure     500
// @Router      /compress/:bucket/*key [get]
func (r *compressionRoutes) compress(cu *gin.Context) {
	ctx, span := otel.Tracer(traceName).Start(cu, "compress-api")
	defer span.End()

	bucket := cu.Param("bucket")
	key := cu.Param("key")
	err := r.cu.PlanCompression(ctx, bucket, key)
	if err != nil {
		r.l.Error(err, "http - v1 - compress")
		errorResponse(cu, http.StatusInternalServerError, "failed to plan compression")
		return
	}

	cu.JSON(http.StatusOK, "{'status':'OK'}")
}

// @Summary     Show history
// @Description Show all translation history
// @ID          history
// @Tags  	    translation
// @Produce     file
// @Success     200
// @Failure     500
// @Router      /compress/:bucket/*key [get]
func (r *compressionRoutes) decompress(cu *gin.Context) {
	ctx, span := otel.Tracer(traceName).Start(cu, "decompress-api")
	defer span.End()

	bucket := cu.Param("bucket")
	key := cu.Param("key")

	content, err := r.cu.GetDecompression(ctx, bucket, key)
	if err != nil {
		r.l.Error(err, "http - v1 - decompress")
		errorResponse(cu, http.StatusInternalServerError, "failed to get decompression")
		return
	}

	contentReader := bytes.NewReader(content)

	http.ServeContent(cu.Writer, cu.Request, key, time.Now(), contentReader)
}
