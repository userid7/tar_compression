package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"

	"audio_compression/config"
)

// Initialize new RabbitMQ connection
func NewRabbitMQConn(cfg *config.Config) (*amqp.Connection, error) {
	connAddr := fmt.Sprintf(cfg.RMQ.URL)
	return amqp.Dial(connAddr)
}
