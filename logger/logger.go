package logger

import "go.uber.org/zap"

var defLogger *zap.Logger

func init() {
	l, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	zap.RedirectStdLog(l)
	l = l.WithOptions(zap.AddCallerSkip(1))
	defLogger = l
}

func Debug(msg string, fields ...zap.Field) {
	defLogger.Debug(msg, fields...)
}
func Info(msg string, fields ...zap.Field) {
	defLogger.Info(msg, fields...)
}
func Warn(msg string, fields ...zap.Field) {
	defLogger.Warn(msg, fields...)
}
func Error(msg string, fields ...zap.Field) {
	defLogger.Error(msg, fields...)
}
