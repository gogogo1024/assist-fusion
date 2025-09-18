package common

import (
	"go.uber.org/zap"
)

var Logger *zap.Logger

func InitLogger() {
	if Logger != nil {
		return
	}
	l, _ := zap.NewProduction()
	Logger = l
}
