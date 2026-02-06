package streaming

import "github.com/nats-io/nats.go"

/*
ConfigureDLQ configures a Dead Letter Subject for a JetStream consumer.
*/
func ConfigureDLQ(js nats.JetStreamContext, subject, dlq string, maxDeliver int) error {
	_, err := js.AddConsumer(subject, &nats.ConsumerConfig{
		AckPolicy:      nats.AckExplicitPolicy,
		MaxDeliver:     maxDeliver,
		DeliverSubject: dlq,
	})
	return err
}
