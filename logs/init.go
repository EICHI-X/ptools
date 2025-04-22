package logs

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/EICHI-X/ptools/env"
	"go.uber.org/zap"

	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {

	var logFilePath string
	dir := "./"
	logFilePath = dir + "/log/app/"
	if err := os.MkdirAll(logFilePath, 0o777); err != nil {
		log.Println(err.Error())

		panic(err.Error())

	}

	// 将文件名设置为日期
	logFileName := time.Now().Format("2006-01-02") + ".log"
	fileName := path.Join(logFilePath, logFileName)
	if _, err := os.Stat(fileName); err != nil {
		if _, err := os.Create(fileName); err != nil {
			log.Println(err.Error())
			panic(err.Error())

		}
	}

	// logger := hertzlogrus.NewLogger()
	// 提供压缩和删除
	lumberjackLogger := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    20,   // 一个文件最大可达 20M。
		MaxBackups: 5,    // 最多同时保存 5 个文件。
		MaxAge:     10,   // 一个文件最多可以保存 10 天。
		Compress:   true, // 用 gzip 压缩。
	}
	// logger.SetOutput(lumberjackLogger)
	// logger.SetLevel(hlog.LevelDebug)
	logger := NewLoggerZap(WithZapOptions(zap.AddCaller(), zap.AddCallerSkip(3)))
	if env.IsPpe() || env.IsBoe() {
		logger.SetLevel(LevelDebug)
	}
	logger.SetOutput(lumberjackLogger)
	// hlog.SetLogger(logger)
	SetLogger(logger) // option with caller
}
