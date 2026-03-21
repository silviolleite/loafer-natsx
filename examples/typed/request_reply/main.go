package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	coreprod "github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/reply"
	"github.com/silviolleite/loafer-natsx/router"
	"github.com/silviolleite/loafer-natsx/typed"
)

// Order represents a typed request message.
type Order struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

// ProcessedOrder represents the response after processing.
type ProcessedOrder struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
}

// ErrInsufficientFunds is a domain error returned when the order amount is invalid.
var ErrInsufficientFunds = errors.New("insufficient funds")

// InsufficientFundsError implements reply.CodedError so the error code
// propagates through the X-Error-Code header automatically.
type InsufficientFundsError struct{}

func (e *InsufficientFundsError) Error() string { return ErrInsufficientFunds.Error() }
func (e *InsufficientFundsError) Code() string  { return "INSUFFICIENT_FUNDS" }

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("typed-request-reply"),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	// Create consumer
	cons, err := consumer.New(nc, logger)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	// Create request-reply route with typed reply function
	replyFn := typed.WrapReply(func(ctx context.Context, result ProcessedOrder, handlerErr error) ([]byte, nats.Header, error) {
		return reply.JSON(ctx, result, handlerErr)
	})

	route, err := router.New(
		router.TypeRequestReply,
		"orders.process",
		router.WithReply(replyFn),
		router.WithQueueGroup("orders-processor"),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	codec := typed.JSONCodec[Order]{}

	// Start consumer with typed handler.
	// When the handler returns an error, reply.JSON encodes it into the
	// response headers (X-Status, X-Error-Code) and body automatically.
	err = cons.Start(ctx, route, typed.WrapHandler(codec, func(ctx context.Context, msg Order) (any, error) {
		fmt.Printf("received request: order %s (%.2f)\n", msg.OrderID, msg.Amount)

		if msg.Amount <= 0 {
			return nil, &InsufficientFundsError{}
		}

		return ProcessedOrder{
			Status:  "processed",
			OrderID: msg.OrderID,
		}, nil
	}))
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	// Create typed requester with an ErrorDecoder that maps reply errors
	// back to domain errors. Without the decoder, errors arrive as *typed.ReplyError.
	strategy := coreprod.NewCoreStrategy(nc)
	reqCodec := typed.JSONCodec[Order]{}
	resCodec := typed.JSONCodec[ProcessedOrder]{}

	decoder := typed.WithErrorDecoder(func(status reply.Status, code string, body []byte) error {
		switch code {
		case "INSUFFICIENT_FUNDS":
			return ErrInsufficientFunds
		default:
			return fmt.Errorf("unexpected error: %s", string(body))
		}
	})

	requester, err := typed.NewRequester(strategy, "orders.process", reqCodec, resCodec, decoder)
	if err != nil {
		slog.Error("failed to create requester", "error", err)
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// 1) Successful request
	result, err := requester.Request(reqCtx, Order{OrderID: "123", Amount: 99.90})
	if err != nil {
		slog.Error("request failed", "error", err)
		return
	}
	fmt.Printf("received reply: %s (order %s)\n", result.Status, result.OrderID)

	// 2) Request that triggers a domain error — the ErrorDecoder reconstructs
	// the original ErrInsufficientFunds so the caller can use errors.Is.
	_, err = requester.Request(reqCtx, Order{OrderID: "456", Amount: -10})
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			fmt.Println("got domain error: insufficient funds")
		}

		// Without ErrorDecoder, you can still inspect the raw reply metadata:
		var re *typed.ReplyError
		if errors.As(err, &re) {
			fmt.Printf("reply error: status=%s code=%s\n", re.Status, re.Code)
		}
	}

	// output:
	// received request: order 123 (99.90)
	// received reply: processed (order 123)
	// received request: order 456 (-10.00)
	// got domain error: insufficient funds
}
