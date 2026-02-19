package consumer

import (
	"context"
	"strconv"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/silviolleite/loafer-natsx/logger"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/router"
)

// Consumer handles message consumption using defined routes and handlers.
type Consumer struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger logger.Logger
}

// New creates a new Consumer instance.
func New(nc *nats.Conn, log logger.Logger) (*Consumer, error) {
	if log == nil {
		log = logger.NopLogger{}
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	return &Consumer{nc: nc, js: js, logger: log}, nil
}

// Start begins consuming messages based on the provided route and handler.
func (p *Consumer) Start(ctx context.Context, route *router.Route, handler HandlerFunc) error {
	switch route.Type() {
	case router.TypePubSub:
		return p.startPubSub(ctx, route, handler)

	case router.TypeQueue:
		return p.startQueue(ctx, route, handler)

	case router.TypeRequestReply:
		return p.startRequestReply(ctx, route, handler)

	case router.TypeJetStream:
		return p.startJetStream(ctx, route, handler)
	}

	return loafernatsx.ErrUnsupportedType
}

func (p *Consumer) startPubSub(ctx context.Context, route *router.Route, handler HandlerFunc) error {
	sub, err := p.nc.Subscribe(route.Subject(), func(msg *nats.Msg) {
		p.safeHandle(ctx, msg.Subject, func() {
			_, err := handler(ctx, msg.Data)
			if err != nil {
				p.logger.Error("handler error", "subject", msg.Subject, "error", err)
			}
		})
	})

	if err != nil {
		return err
	}

	p.drainOnCancel(ctx, func() { _ = sub.Drain() })

	return nil
}

func (p *Consumer) startQueue(ctx context.Context, route *router.Route, handler HandlerFunc) error {
	sub, err := p.nc.QueueSubscribe(route.Subject(), route.QueueGroup(), func(msg *nats.Msg) {
		p.safeHandle(ctx, msg.Subject, func() {
			_, err := handler(ctx, msg.Data)
			if err != nil {
				p.logger.Error("handler error", "subject", msg.Subject, "error", err)
			}
		})
	})
	if err != nil {
		return err
	}

	p.drainOnCancel(ctx, func() { _ = sub.Drain() })

	return nil
}

func (p *Consumer) startRequestReply(ctx context.Context, route *router.Route, handler HandlerFunc) error {
	sub, err := p.nc.QueueSubscribe(route.Subject(), route.QueueGroup(), func(msg *nats.Msg) {
		p.safeHandle(ctx, msg.Subject, func() {
			p.handleRequestReplyMessage(ctx, route, handler, msg)
		})
	})
	if err != nil {
		return err
	}

	p.drainOnCancel(ctx, func() { _ = sub.Drain() })

	return nil
}

func (p *Consumer) handleRequestReplyMessage(
	ctx context.Context,
	route *router.Route,
	handler HandlerFunc,
	msg *nats.Msg,
) {
	result, hErr := handler(ctx, msg.Data)

	if route.ReplyFunc() == nil {
		p.defaultReply(msg, hErr)
		return
	}

	data, headers, rErr := route.ReplyFunc()(ctx, result, hErr)
	if rErr != nil {
		p.logger.Error("reply builder error", "subject", msg.Subject, "error", rErr)
		return
	}

	out := &nats.Msg{
		Subject: msg.Reply,
		Data:    data,
		Header:  headers,
	}

	propagateHeaders(msg, out)

	if err := p.nc.PublishMsg(out); err != nil {
		p.logger.Error("reply send error", "subject", msg.Subject, "error", err)
	}
}

func (p *Consumer) defaultReply(msg *nats.Msg, err error) {
	if err != nil {
		p.logger.Error("handler error", "subject", msg.Subject, "error", err)
		return
	}

	if rErr := msg.Respond([]byte("ok")); rErr != nil {
		p.logger.Error("reply send error", "subject", msg.Subject, "error", rErr)
	}
}

func (p *Consumer) startJetStream(ctx context.Context, route *router.Route, handler HandlerFunc) error {
	consumerCfg := jetstream.ConsumerConfig{
		Durable:       route.Durable(),
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverPolicy(route.DeliveryPolicy()),
		MaxDeliver:    route.MaxDeliver(),
		AckWait:       route.AckWait(),
		FilterSubject: route.Subject(),
	}

	cons, err := p.js.CreateOrUpdateConsumer(ctx, route.Stream(), consumerCfg)
	if err != nil {
		return err
	}

	consumeCtx, err := cons.Consume(func(msg jetstream.Msg) {
		p.safeHandle(ctx, route.Subject(), func() {
			p.handleJetStreamMessage(ctx, route, handler, msg)
		})
	})
	if err != nil {
		return err
	}

	p.drainOnCancel(ctx, func() { consumeCtx.Stop() })

	return nil
}

func (p *Consumer) handleJetStreamMessage(
	ctx context.Context,
	route *router.Route,
	handler HandlerFunc,
	msg jetstream.Msg,
) {
	meta, _ := msg.Metadata()
	_, hErr := handler(ctx, msg.Data())
	if hErr != nil {
		p.handleJetStreamError(route, msg, meta, hErr)
		return
	}

	if err := msg.Ack(); err != nil {
		p.logger.Error("ack error", "subject", route.Subject(), "error", err)
	}
}

func (p *Consumer) handleJetStreamError(
	route *router.Route,
	msg jetstream.Msg,
	meta *jetstream.MsgMetadata,
	err error,
) {
	p.logger.Error("handler error", "subject", route.Subject(), "error", err)

	if route.DLQEnabled() && int(meta.NumDelivered) >= route.MaxDeliver() {
		p.publishToDLQ(route, msg, meta, err)
		return
	}

	if nakErr := msg.Nak(); nakErr != nil {
		p.logger.Error("nak error", "subject", route.Subject(), "error", nakErr)
	}
}

func (p *Consumer) publishToDLQ(
	route *router.Route,
	msg jetstream.Msg,
	meta *jetstream.MsgMetadata,
	err error,
) {
	const dlqPRefix = "dlq."
	headers := nats.Header{}
	headers.Set(HeaderErrorKey, err.Error())
	headers.Set(HeaderRetryCountKey, strconv.Itoa(int(meta.NumDelivered)))

	dlqSubject := dlqPRefix + route.Subject()

	if pubErr := p.nc.PublishMsg(&nats.Msg{
		Subject: dlqSubject,
		Header:  headers,
		Data:    msg.Data(),
	}); pubErr != nil {
		p.logger.Error("dlq publish error", "subject", dlqSubject, "error", pubErr)
		_ = msg.Nak()
		return
	}

	if ackErr := msg.Ack(); ackErr != nil {
		p.logger.Error("ack error after dlq", "subject", route.Subject(), "error", ackErr)
	}
}

func (p *Consumer) drainOnCancel(ctx context.Context, fn func()) {
	go func() {
		<-ctx.Done()
		fn()
	}()
}

func (p *Consumer) safeHandle(
	ctx context.Context,
	subject string,
	fn func(),
) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error(
				"handler panic recovered",
				"subject", subject,
				"panic", r,
			)
		}
	}()

	fn()
}
