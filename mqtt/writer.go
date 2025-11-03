package mqtt

import (
	"context"
)

// Writer is the minimum abstraction around writing values to MQTT.
type Writer interface {
	// WriteTopic writes the provided value to the specified topic with the specified WriteOptions.
	WriteTopic(ctx context.Context, topic string, options WriteOptions, value []byte) error
}

// Error discards the result of Writer.WriteTopic, returning just the error. Used to join multiple errors when you don't
// care about returned values.
func Error[T any](_ T, err error) error {
	return err
}
