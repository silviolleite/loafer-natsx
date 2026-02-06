package producer

import "time"

type config struct {
	subject     string
	fifo        bool
	dedupID     string
	dedupWindow time.Duration
}
