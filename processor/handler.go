package processor

import "context"

/*
HandlerFunc defines the function signature for message processing.
*/
type HandlerFunc func(ctx context.Context, data []byte) (any, error)
