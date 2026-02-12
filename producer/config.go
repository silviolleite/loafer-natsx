package producer

import "github.com/silviolleite/loafer-natsx/logger"

type config struct {
	subject string
	useJS   bool
	log     logger.Logger
}
