package zap

import (
	"context"
	"log"
	"os"
	"path"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"gopkg.in/natefinch/lumberjack.v2"
)
func init(){
    var logFilePath string
	dir := "./"
	logFilePath = dir + "/logs/"
	if err := os.MkdirAll(logFilePath, 0o777); err != nil {
		log.Println(err.Error())
		return
	}

	// 将文件名设置为日期
	logFileName := time.Now().Format("2006-01-02") + ".log"
	fileName := path.Join(logFilePath, logFileName)
	if _, err := os.Stat(fileName); err != nil {
		if _, err := os.Create(fileName); err != nil {
			log.Println(err.Error())
			return
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
	
	// logus := kitexlogrus.NewLogger()
	  klog.SetLogger(NewLogger())
	// klog.SetLogger(logus)
	klog.SetLevel(klog.LevelDebug)
	klog.SetOutput(lumberjackLogger)
	klog.CtxInfof(context.TODO(), "init")
}