package rmq

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"

	"audio_compression/config"
	"audio_compression/entity"
	"audio_compression/internal/compression"
	"audio_compression/internal/storage/s3repo"
	"audio_compression/pkg/logger"
	"audio_compression/pkg/rabbitmq"
)

type AMQPWorker struct {
	amqpChan        *amqp.Channel
	cfg             *config.Config
	l               *logger.Logger
	blobStorageRepo entity.StorageRepository
	cu              *compression.CompressionUsecase
}

// NewCompressionConsumer Emails rabbitmq Consumer constructor
func NewAMQPWorker(cfg *config.Config, l *logger.Logger, cu *compression.CompressionUsecase) (*AMQPWorker, error) {
	mqConn, err := rabbitmq.NewRabbitMQConn(cfg)
	if err != nil {
		return nil, err
	}
	amqpChan, err := mqConn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "amqpw.amqpConn.Channel")
	}
	s3Repo, err := s3repo.NewS3Repository()
	if err != nil {
		l.Error(err)
		l.Fatal("Failed to init S3 Repository")
	}

	return &AMQPWorker{cfg: cfg, l: l, amqpChan: amqpChan, cu: cu, blobStorageRepo: s3Repo}, nil
}

// SetupExchangeAndQueue create exchange and queue
func (amqpw *AMQPWorker) SetupExchangeAndQueue(exchange, queueName, bindingKey, consumerTag string) error {
	amqpw.l.Info("Declaring exchange: %s", exchange)
	err := amqpw.amqpChan.ExchangeDeclare(
		exchange,
		exchangeKind,
		exchangeDurable,
		exchangeAutoDelete,
		exchangeInternal,
		exchangeNoWait,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "Error ch.ExchangeDeclare")
	}

	queue, err := amqpw.amqpChan.QueueDeclare(
		queueName,
		queueDurable,
		queueAutoDelete,
		queueExclusive,
		queueNoWait,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "Error ch.QueueDeclare")
	}

	amqpw.l.Info("Declared queue, binding it to exchange: Queue: %v, messageCount: %v, "+
		"consumerCount: %v, exchange: %v, exchange: %v, bindingKey: %v",
		queue.Name,
		queue.Messages,
		queue.Consumers,
		exchange,
		bindingKey,
	)

	err = amqpw.amqpChan.QueueBind(
		queue.Name,
		bindingKey,
		exchange,
		queueNoWait,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "Error ch.QueueBind")
	}

	amqpw.l.Info("Queue bound to exchange, starting to consume from queue, consumerTag: %v", consumerTag)
	return nil
}

// CloseChan Close messages chan
func (amqpw *AMQPWorker) CloseChan() error {
	if err := amqpw.amqpChan.Close(); err != nil {
		amqpw.l.Error("AMQPWorker CloseChan: %v", err)
		return err
	}
	return nil
}

// Publish message
func (amqpw *AMQPWorker) Publish(exchange, key, contentType, corrId, replyTo string, body []byte) error {

	amqpw.l.Info("Publishing message Exchange: %s, RoutingKey: %s", exchange, key)

	if err := amqpw.amqpChan.Publish(
		exchange,
		key,
		publishMandatory,
		publishImmediate,
		amqp.Publishing{
			ContentType:   contentType,
			DeliveryMode:  amqp.Persistent,
			MessageId:     uuid.New().String(),
			Timestamp:     time.Now(),
			CorrelationId: corrId,
			ReplyTo:       replyTo,
			Body:          body,
		},
	); err != nil {
		return errors.Wrap(err, "ch.Publish")
	}

	return nil
}

// StartConsumer Start new rabbitmq consumer
func (c *AMQPWorker) StartConsumer() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := c.amqpChan
	compressionQueue := "compress_request"
	decompressionQueue := "decompress_request"

	if err := c.SetupExchangeAndQueue("audio_compression", compressionQueue, "compress", ""); err != nil {
		return errors.Wrap(err, "SetupExchangeAndQueue")
	}

	if err := c.SetupExchangeAndQueue("audio_compression", decompressionQueue, "decompress", ""); err != nil {
		return errors.Wrap(err, "SetupExchangeAndQueue")
	}

	compressionDeliveries, err := ch.Consume(
		compressionQueue,
		"",
		consumeAutoAck,
		consumeExclusive,
		consumeNoLocal,
		consumeNoWait,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "Consume")
	}

	decompressionDeliveries, err := ch.Consume(
		decompressionQueue,
		"",
		consumeAutoAck,
		consumeExclusive,
		consumeNoLocal,
		consumeNoWait,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "Consume")
	}

	go c.ConsumeCompression(ctx, compressionDeliveries)
	go c.ConsumeDecompression(ctx, decompressionDeliveries)

	chanErr := <-ch.NotifyClose(make(chan *amqp.Error))
	c.l.Error("ch.NotifyClose: %v", chanErr)
	return chanErr
}

func (c *AMQPWorker) ConsumeCompression(_ context.Context, messages <-chan amqp.Delivery) {
	for delivery := range messages {
		ctx := context.Background()
		ctx, span := otel.Tracer(traceName).Start(ctx, "consumer")
		defer span.End()

		var compressionRequest entity.CompressionRequest

		if err := json.Unmarshal(delivery.Body, &compressionRequest); err != nil {
			c.l.Error(err)
			time.Sleep(time.Duration(time.Second * 5))
			continue
		}

		err, shouldRetry := c.cu.DoCompression(ctx, compressionRequest.Bucket, compressionRequest.Key)
		if err != nil {
			c.l.Error(err)
			if shouldRetry {
				delivery.Reject(true)
			} else {
				delivery.Ack(false)
			}
			time.Sleep(time.Duration(time.Second * 5))
			continue
		}
		delivery.Ack(false)

	}
}
func (c *AMQPWorker) ConsumeDecompression(_ context.Context, messages <-chan amqp.Delivery) {
	for delivery := range messages {
		ctx := context.Background()
		ctx, span := otel.Tracer(traceName).Start(ctx, "consumer")
		defer span.End()

		var compressionRequest entity.CompressionRequest

		if err := json.Unmarshal(delivery.Body, &compressionRequest); err != nil {
			c.l.Error(err)
			delivery.Ack(true)
			continue
		}

		result, err := c.cu.GetDecompression(ctx, compressionRequest.Bucket, compressionRequest.Key)
		if err != nil {
			c.l.Error(err)
			delivery.Reject(false)
			continue
		}

		fileName, err := WriteByteToFileSystem(result)
		if err != nil {
			c.l.Error(err)
			delivery.Reject(false)
			continue
		}

		compressionResponse := &entity.CompressionResponse{ResultAddress: fileName, ResultType: "FS"}

		s, err := json.Marshal(compressionResponse)
		if err != nil {
			c.l.Error(err)
			delivery.Ack(true)
			continue
		}

		c.Publish("audio_compression", "decompression_response", "application/json", delivery.CorrelationId, "", s)
		delivery.Ack(false)
	}
}

func WriteByteToFileSystem(b []byte) (string, error) {
	f, err := os.CreateTemp(os.TempDir(), "decompress-")
	if err != nil {
		return "", err
	}
	defer f.Close()

	fileName := f.Name()

	if _, err := f.Write(b); err != nil {
		return "", err
	}

	return fileName, nil
}
