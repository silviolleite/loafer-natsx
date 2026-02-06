package processor

import (
	"context"
	"strconv"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/consumer"
)

/*
Processor handles message consumption using defined routes and handlers.
*/
type Processor struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger Logger
}

/*
New creates a new Processor instance.
*/
func New(nc *nats.Conn, logger Logger) (*Processor, error) {
	if logger == nil {
		logger = nopLogger{}
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	return &Processor{nc: nc, js: js, logger: logger}, nil
}

/*
Start begins consuming messages based on the provided route and handler.
*/
func (p *Processor) Start(ctx context.Context, route *consumer.Route, handler HandlerFunc) error {
	switch route.Type() {
	case consumer.RouteTypePubSub:
		_, err := p.nc.Subscribe(route.Subject(), func(msg *nats.Msg) {
			_, err := handler(ctx, msg.Data)
			if err != nil {
				p.logger.Error("handler error", "subject", msg.Subject, "err", err)
			}
		})
		return err

	case consumer.RouteTypeQueue:
		_, err := p.nc.QueueSubscribe(route.Subject(), route.QueueGroup(), func(msg *nats.Msg) {
			_, err := handler(ctx, msg.Data)
			if err != nil {
				p.logger.Error("handler error", "subject", msg.Subject, "err", err)
			}
		})
		return err

	case consumer.RouteTypeRequestReply:
		_, err := p.nc.Subscribe(route.Subject(), func(msg *nats.Msg) {
			result, hErr := handler(ctx, msg.Data)

			if route.ReplyFunc() == nil {
				if hErr != nil {
					p.logger.Error("handler error", "subject", msg.Subject, "err", hErr)
					return
				}
				if err := msg.Respond([]byte("ok")); err != nil {
					p.logger.Error("reply send error", "subject", msg.Subject, "err", err)
				}
				return
			}

			data, headers, rErr := route.ReplyFunc()(ctx, result, hErr)
			if rErr != nil {
				p.logger.Error("reply builder error", "subject", msg.Subject, "err", rErr)
				return
			}

			out := &nats.Msg{
				Subject: msg.Reply,
				Data:    data,
				Header:  headers,
			}

			propagateHeaders(msg, out)

			if err := p.nc.PublishMsg(out); err != nil {
				p.logger.Error("reply send error", "subject", msg.Subject, "err", err)
			}
		})
		return err

	case consumer.RouteTypeJetStream:
		consumerCfg := jetstream.ConsumerConfig{
			Durable:       route.Durable(),
			AckPolicy:     jetstream.AckExplicitPolicy,
			MaxDeliver:    route.MaxDeliver(),
			AckWait:       route.AckWait(),
			FilterSubject: route.Subject(),
		}

		cons, err := p.js.CreateOrUpdateConsumer(ctx, route.Stream(), consumerCfg)
		if err != nil {
			return err
		}

		_, err = cons.Consume(func(msg jetstream.Msg) {
			meta, _ := msg.Metadata()
			_, hErr := handler(ctx, msg.Data())

			if hErr != nil {
				p.logger.Error("handler error", "subject", route.Subject(), "err", hErr)

				if route.DLQEnabled() && int(meta.NumDelivered) >= route.MaxDeliver() {
					headers := nats.Header{}
					headers.Set("X-Error", hErr.Error())
					headers.Set("X-Retry-Count", strconv.Itoa(int(meta.NumDelivered)))

					dlqSubject := "dlq." + route.Subject()
					if pubErr := p.nc.PublishMsg(&nats.Msg{
						Subject: dlqSubject,
						Header:  headers,
						Data:    msg.Data(),
					}); pubErr != nil {
						p.logger.Error("dlq publish error", "subject", dlqSubject, "err", pubErr)
						_ = msg.Nak()
						return
					}

					if ackErr := msg.Ack(); ackErr != nil {
						p.logger.Error("ack error after dlq", "subject", route.Subject(), "err", ackErr)
					}
					return
				}

				if nakErr := msg.Nak(); nakErr != nil {
					p.logger.Error("nak error", "subject", route.Subject(), "err", nakErr)
				}
				return
			}

			if ackErr := msg.Ack(); ackErr != nil {
				p.logger.Error("ack error", "subject", route.Subject(), "err", ackErr)
			}
		})

		return err
	}

	return loafernastx.ErrUnsupportedType
}
