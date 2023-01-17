package logger

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logLevel = zap.NewAtomicLevel()
	log      *zap.SugaredLogger
)

func InitLogger(filePath string) {
	fmt.Println(filePath)
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:  filePath,
		MaxSize:   512, // MB
		LocalTime: true,
		Compress:  true,
	})

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		w,
		logLevel,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	log = logger.Sugar()
}

func NewLogger(filePath string) *zap.SugaredLogger {
	InitLogger(filePath)
	return log
}
