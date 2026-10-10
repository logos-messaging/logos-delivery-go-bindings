package kernel

import (
	"sync"

	"go.uber.org/zap"
)

var (
	once  sync.Once
	sugar *zap.SugaredLogger
)

func _getLogger() *zap.SugaredLogger {
	once.Do(func() {

		config := zap.NewDevelopmentConfig()
		l, err := config.Build()
		if err != nil {
			panic(err)
		}
		sugar = l.Sugar()
	})
	return sugar
}

func SetLogger(newLogger *zap.Logger) {
	once.Do(func() {})

	sugar = newLogger.Sugar()
}

func logDebug(msg string, args ...interface{}) {
	_getLogger().Debugf(msg, args...)
}

//nolint:unused // completes the level set alongside logDebug/logWarn/logError.
func logInfo(msg string, args ...interface{}) {
	_getLogger().Infof(msg, args...)
}

func logWarn(msg string, args ...interface{}) {
	_getLogger().Warnf(msg, args...)
}

func logError(msg string, args ...interface{}) {
	_getLogger().Errorf(msg, args...)
}
