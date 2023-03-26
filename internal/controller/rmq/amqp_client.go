package rmq

import (
	"audio_compression/config"
	"audio_compression/entity"
	"audio_compression/pkg/logger"
	"audio_compression/pkg/rabbitmq"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type AMQPClient struct {
	amqpChan   *amqp.Channel
	cfg        *config.Config
	l          *logger.Logger
	compClient *DecompressionClient
}

var replyTos = "decompression_response"

// NewCompressionConsumer Emails rabbitmq Consumer constructor
func NewAMQPClient(cfg *config.Config, l *logger.Logger) (*AMQPClient, error) {
	mqConn, err := rabbitmq.NewRabbitMQConn(cfg)
	if err != nil {
		return nil, err
	}
	amqpChan, err := mqConn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "amqpw.amqpConn.Channel")
	}
	compClient := NewDecompressionClient(cfg, l)

	c := &AMQPClient{cfg: cfg, l: l, amqpChan: amqpChan, compClient: compClient}

	if c.SetupExchangeAndQueue("audio_compression", "decompress_response", "decompression_response", ""); err != nil {
		l.Error(err)
		l.Fatal("Failed to setup exchange and queue")
	}

	return c, nil
}

// SetupExchangeAndQueue create exchange and queue
func (amqpw *AMQPClient) SetupExchangeAndQueue(exchange, queueName, bindingKey, consumerTag string) error {
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
func (amqpw *AMQPClient) CloseChan() error {
	if err := amqpw.amqpChan.Close(); err != nil {
		amqpw.l.Error("AMQPClient CloseChan: %v", err)
		return err
	}
	return nil
}

// Publish message
func (amqpw *AMQPClient) Publish(exchange, key, contentType, corrId, replyTo string, body []byte) error {

	amqpw.l.Info("Publishing message Exchange: %s, RoutingKey: %s", amqpw.cfg.RMQ.ServerExchange, "")

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

func (c *AMQPClient) DecompressionConsumer() error {
	ch := c.amqpChan

	deliveries, err := ch.Consume(
		"decompress_response",
		"",
		consumeAutoAck,
		consumeExclusive,
		consumeNoLocal,
		consumeNoWait,
		nil,
	)
	if err != nil {
		c.l.Fatal(err)
		return err
	}

	for d := range deliveries {
		c.l.Info("receive decompression response")
		var compressionResponse entity.CompressionResponse
		if err := json.Unmarshal(d.Body, &compressionResponse); err != nil {
			c.l.Error(err)
			time.Sleep(time.Duration(time.Second * 3))
			d.Ack(false)
			continue
		}
		c.compClient.SetDecompressionResponse(d.CorrelationId, compressionResponse)
		d.Ack(false)
	}

	chanErr := <-ch.NotifyClose(make(chan *amqp.Error))
	c.l.Error("ch.NotifyClose: %v", chanErr)
	return chanErr
}

// func (p *AMQPClient) PlanCompression(ctx context.Context, bucket, key string) error {
// 	payload := entity.CompressionRequest{Bucket: bucket, Key: key}
// 	s, err := json.Marshal(payload)
// 	if err != nil {
// 		return err
// 	}
// 	if err := p.Publish("audio_compression", "compress", "application/json", "", "", s); err != nil {
// 		return err
// 	}
// 	return nil
// }

func (p *AMQPClient) CallCompressionApi(ctx context.Context, bucket, key, compType, corrId, replyTo string) error {
	payload := entity.CompressionRequest{Bucket: bucket, Key: key, Type: compType}
	s, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	switch compType {
	case "compress":
		if err := p.Publish("audio_compression", "compress", "application/json", corrId, replyTo, s); err != nil {
			return err
		}
	case "decompress":
		if err := p.Publish("audio_compression", "decompress", "application/json", corrId, replyTo, s); err != nil {
			return err
		}
	}

	return nil
}

func (cs *AMQPClient) PlanCompression(ctx context.Context, bucket, key string) error {
	corrId, isAlreadyExist := cs.compClient.GetOrCreateRequest(bucket, key, "compress")

	if !isAlreadyExist {
		if err := cs.CallCompressionApi(ctx, bucket, key, "compress", corrId, "compression_response"); err != nil {
			return err
		}
	}
	return nil
}

func (cs *AMQPClient) GetDecompression(ctx context.Context, bucket, key string) ([]byte, error) {
	corrId, isAlreadyExist := cs.compClient.GetOrCreateRequest(bucket, key, "decompress")

	if !isAlreadyExist {
		if err := cs.CallCompressionApi(ctx, bucket, key, "decompress", corrId, "decompression_response"); err != nil {
			return nil, err
		}
	}
	res, err := cs.compClient.GetDecompressionResponse(ctx, corrId, 100)
	if err != nil {
		return nil, err
	}

	fmt.Println(res.ResultType)
	fmt.Println(res.ResultAddress)

	resultByte, err := cs.compClient.GetByteFromFileSystem(res.ResultAddress)
	if err != nil {
		return nil, err
	}

	return resultByte, nil
}
