package producer

import "time"

/*
Config defines the behavior of a message producer.
*/
type Config struct {
	Subject     string
	FIFO        bool
	DedupID     string
	DedupWindow time.Duration
}
