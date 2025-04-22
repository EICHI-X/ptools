// Copyright 2022 CloudWeGo Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs

import (
	"context"
	"errors"
	"fmt"
	"io"


	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

)

// var _ FullLoggerZap = (*LoggerZap)(nil)

// Ref to https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/README.md#json-formats
const (
	traceIDKey    = "trace_id"
	spanIDKey     = "span_id"
	traceFlagsKey = "trace_flags"
	logEventKey   = "log"
)

var (
	logSeverityTextKey = attribute.Key("otel.log.severity.text")
	logMessageKey      = attribute.Key("otel.log.message")
)

type LoggerZap struct {
	*zap.SugaredLogger
	config *config
}

var logger FullLogger 

// SetOutput sets the output of default logger. By default, it is stderr.
func SetOutput(w io.Writer) {
	logger.SetOutput(w)
}

// SetLevel sets the level of logs below which logs will not be output.
// The default log level is LevelTrace.
// Note that this method is not concurrent-safe.
func SetLevel(lv Level) {
	logger.SetLevel(lv)
}

// DefaultLogger return the default logger for kitex.
func DefaultLogger() FullLogger {
	return logger
}

// SetLogger sets the default logger.
// Note that this method is not concurrent-safe and must not be called
// after the use of DefaultLogger and global functions in this package.
func SetLogger(v FullLogger) {
	logger = v
}
func GetLogger()FullLogger{
	return logger
}
func NewLoggerZap(opts ...Option) *LoggerZap {
	config := defaultConfig()

	// apply options
	for _, opt := range opts {
		opt.apply(config)
	}

	logger := zap.New(
		zapcore.NewCore(config.coreConfig.enc, config.coreConfig.ws, config.coreConfig.lvl),
		config.zapOpts...)

	return &LoggerZap{
		SugaredLogger: logger.Sugar(),
		config:        config,
	}
}

func (l *LoggerZap) Log(level Level, kvs ...interface{}) {
	logger := l.With()
	switch level {
	case LevelTrace, LevelDebug:
		logger.Debug(kvs...)
	case LevelInfo:
		logger.Info(kvs...)
	case LevelNotice, LevelWarn:
		logger.Warn(kvs...)
	case LevelError:
		logger.Error(kvs...)
	case LevelFatal:
		logger.Fatal(kvs...)
	default:
		logger.Warn(kvs...)
	}
}

func (l *LoggerZap) Logf(level Level, format string, kvs ...interface{}) {
	logger := l.With()
	switch level {
	case LevelTrace, LevelDebug:
		logger.Debugf(format, kvs...)
	case LevelInfo:
		logger.Infof(format, kvs...)
	case LevelNotice, LevelWarn:
		logger.Warnf(format, kvs...)
	case LevelError:
		logger.Errorf(format, kvs...)
	case LevelFatal:
		logger.Fatalf(format, kvs...)
	default:
		logger.Warnf(format, kvs...)
	}
}

func (l *LoggerZap) CtxLogf(level Level, ctx context.Context, format string, kvs ...interface{}) {
	var zlevel zapcore.Level
	var sl *zap.SugaredLogger

	span := trace.SpanFromContext(ctx)
	var traceKVs []interface{}
	if span.SpanContext().TraceID().IsValid() {
		traceKVs = append(traceKVs, traceIDKey, span.SpanContext().TraceID())
	}
	if span.SpanContext().SpanID().IsValid() {
		traceKVs = append(traceKVs, spanIDKey, span.SpanContext().SpanID())
	}
	if span.SpanContext().TraceFlags().IsSampled() {
		traceKVs = append(traceKVs, traceFlagsKey, span.SpanContext().TraceFlags())
	}
	if len(traceKVs) > 0 {
		sl = l.With(traceKVs...)
	} else {
		sl = l.With()
	}

	switch level {
	case LevelDebug, LevelTrace:
		zlevel = zap.DebugLevel
		sl.Debugf(format, kvs...)
	case LevelInfo:
		zlevel = zap.InfoLevel
		sl.Infof(format, kvs...)
	case LevelNotice, LevelWarn:
		zlevel = zap.WarnLevel
		sl.Warnf(format, kvs...)
	case LevelError:
		zlevel = zap.ErrorLevel
		sl.Errorf(format, kvs...)
	case LevelFatal:
		zlevel = zap.FatalLevel
		sl.Fatalf(format, kvs...)
	default:
		zlevel = zap.WarnLevel
		sl.Warnf(format, kvs...)
	}

	if !span.IsRecording() {
		return
	}

	msg := getMessage(format, kvs)

	attrs := []attribute.KeyValue{
		logMessageKey.String(msg),
		logSeverityTextKey.String(OtelSeverityText(zlevel)),
	}
	span.AddEvent(logEventKey, trace.WithAttributes(attrs...))

	// set span status
	if zlevel <= l.config.traceConfig.errorSpanLevel {
		span.SetStatus(codes.Error, msg)
		span.RecordError(errors.New(msg), trace.WithStackTrace(l.config.traceConfig.recordStackTraceInSpan))
	}
}
func GetTraceId(ctx context.Context)string{
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().TraceID().IsValid() {
		return fmt.Sprint(span.SpanContext().TraceID())
	}
	return ""
}
func (l *LoggerZap) Trace(v ...interface{}) {
	l.Log(LevelTrace, v...)
}

func (l *LoggerZap) Debug(v ...interface{}) {
	l.Log(LevelDebug, v...)
}

func (l *LoggerZap) Info(v ...interface{}) {
	l.Log(LevelInfo, v...)
}

func (l *LoggerZap) Notice(v ...interface{}) {
	l.Log(LevelNotice, v...)
}

func (l *LoggerZap) Warn(v ...interface{}) {
	l.Log(LevelWarn, v...)
}

func (l *LoggerZap) Error(v ...interface{}) {
	l.Log(LevelError, v...)
}

func (l *LoggerZap) Fatal(v ...interface{}) {
	l.Log(LevelFatal, v...)
}

func (l *LoggerZap) Tracef(format string, v ...interface{}) {
	l.Logf(LevelTrace, format, v...)
}

func (l *LoggerZap) Debugf(format string, v ...interface{}) {
	l.Logf(LevelDebug, format, v...)
}

func (l *LoggerZap) Infof(format string, v ...interface{}) {
	l.Logf(LevelInfo, format, v...)
}

func (l *LoggerZap) Noticef(format string, v ...interface{}) {
	l.Logf(LevelInfo, format, v...)
}

func (l *LoggerZap) Warnf(format string, v ...interface{}) {
	l.Logf(LevelWarn, format, v...)
}

func (l *LoggerZap) Errorf(format string, v ...interface{}) {
	l.Logf(LevelError, format, v...)
}

func (l *LoggerZap) Fatalf(format string, v ...interface{}) {
	l.Logf(LevelFatal, format, v...)
}

func (l *LoggerZap) CtxTracef(ctx context.Context, format string, v ...interface{}) {
	l.CtxLogf(LevelDebug, ctx, format, v...)
}

func (l *LoggerZap) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	l.CtxLogf(LevelDebug, ctx, format, v...)
}

func (l *LoggerZap) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	l.CtxLogf(LevelInfo, ctx, format, v...)
}

func (l *LoggerZap) CtxNoticef(ctx context.Context, format string, v ...interface{}) {
	l.CtxLogf(LevelWarn, ctx, format, v...)
}

func (l *LoggerZap) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	l.CtxLogf(LevelWarn, ctx, format, v...)
}

func (l *LoggerZap) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	l.CtxLogf(LevelError, ctx, format, v...)
}

func (l *LoggerZap) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	l.CtxLogf(LevelFatal, ctx, format, v...)
}

func (l *LoggerZap) SetLevel(level Level) {
	var lvl zapcore.Level
	switch level {
	case LevelTrace, LevelDebug:
		lvl = zap.DebugLevel
	case LevelInfo:
		lvl = zap.InfoLevel
	case LevelWarn, LevelNotice:
		lvl = zap.WarnLevel
	case LevelError:
		lvl = zap.ErrorLevel
	case LevelFatal:
		lvl = zap.FatalLevel
	default:
		lvl = zap.WarnLevel
	}
	l.config.coreConfig.lvl.SetLevel(lvl)
}

func (l *LoggerZap) SetOutput(writer io.Writer) {
	ws := zapcore.AddSync(writer)
	log := zap.New(
		zapcore.NewCore(l.config.coreConfig.enc, ws, l.config.coreConfig.lvl),
		l.config.zapOpts...,
	)
	l.config.coreConfig.ws = ws
	l.SugaredLogger = log.Sugar()
}

func (l *LoggerZap) CtxKVLog(ctx context.Context, level Level, format string, kvs ...interface{}) {
	if len(kvs) == 0 || len(kvs)%2 != 0 {
		l.Warn(fmt.Sprint("Keyvalues must appear in pairs:", kvs))
		return
	}

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().TraceID().IsValid() {
		kvs = append(kvs, traceIDKey, span.SpanContext().TraceID())
	}
	if span.SpanContext().SpanID().IsValid() {
		kvs = append(kvs, spanIDKey, span.SpanContext().SpanID())
	}
	if span.SpanContext().TraceFlags().IsSampled() {
		kvs = append(kvs, traceFlagsKey, span.SpanContext().TraceFlags())
	}

	var zlevel zapcore.Level
	zl := l.With()
	switch level {
	case LevelDebug, LevelTrace:
		zlevel = zap.DebugLevel
		zl.Debugw(format, kvs...)
	case LevelInfo:
		zlevel = zap.InfoLevel
		zl.Infow(format, kvs...)
	case LevelNotice, LevelWarn:
		zlevel = zap.WarnLevel
		zl.Warnw(format, kvs...)
	case LevelError:
		zlevel = zap.ErrorLevel
		zl.Errorw(format, kvs...)
	case LevelFatal:
		zlevel = zap.FatalLevel
		zl.Fatalw(format, kvs...)
	default:
		zlevel = zap.WarnLevel
		zl.Warnw(format, kvs...)
	}

	if !span.IsRecording() {
		return
	}

	msg := getMessage(format, kvs)
	attrs := []attribute.KeyValue{
		logMessageKey.String(msg),
		logSeverityTextKey.String(OtelSeverityText(zlevel)),
	}

	// notice: AddEvent,SetStatus,RecordError all have check span.IsRecording
	span.AddEvent(logEventKey, trace.WithAttributes(attrs...))

	// set span status
	if zlevel <= l.config.traceConfig.errorSpanLevel {
		span.SetStatus(codes.Error, msg)
		span.RecordError(errors.New(msg), trace.WithStackTrace(l.config.traceConfig.recordStackTraceInSpan))
	}
}
