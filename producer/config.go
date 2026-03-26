package producer

import (
	"time"

	"github.com/silviolleite/loafer-natsx/logger"
)

const defaultRequestTimeout = 10 * time.Second

type config struct {
	log            logger.Logger
	subject        string
	requestTimeout time.Duration
}
