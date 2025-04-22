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
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Option interface {
	apply(cfg *config)
}

type option func(cfg *config)

func (fn option) apply(cfg *config) {
	fn(cfg)
}

type coreConfig struct {
	enc zapcore.Encoder
	ws  zapcore.WriteSyncer
	lvl zap.AtomicLevel
}

type traceConfig struct {
	recordStackTraceInSpan bool
	errorSpanLevel         zapcore.Level
}

type config struct {
	coreConfig  coreConfig
	zapOpts     []zap.Option
	traceConfig *traceConfig
}

// defaultCoreConfig default zapcore config: json encoder, atomic level, stdout write syncer
func defaultCoreConfig() *coreConfig {
	// default log encoder
	con := zap.NewProductionEncoderConfig()
	con.CallerKey = "line"
	con.FunctionKey = "func"
	con.EncodeLevel =  zapcore.CapitalLevelEncoder
	con.EncodeLevel =  zapcore.CapitalLevelEncoder
	con.EncodeName = zapcore.FullNameEncoder
	con.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	enc := zapcore.NewJSONEncoder(con)
	// default log level
	lvl := zap.NewAtomicLevelAt(zap.InfoLevel)
	// default write syncer stdout
	ws := zapcore.AddSync(os.Stdout)
	// encoderConfig := zapcore.EncoderConfig{
	// 	TimeKey:        "time",
	// 	LevelKey:       "level",
	// 	NameKey:        "name",
	// 	CallerKey:      "line",
	// 	MessageKey:     "msg",
	// 	FunctionKey:    "func",
	// 	StacktraceKey:  "stacktrace",
	// 	LineEnding:     zapcore.DefaultLineEnding,
	// 	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	// 	EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
	// 	EncodeDuration: zapcore.SecondsDurationEncoder,
	// 	EncodeCaller:   zapcore.FullCallerEncoder,
	// 	EncodeName:     zapcore.FullNameEncoder,
	// }

	return &coreConfig{
		enc: enc,
		ws:  ws,
		lvl: lvl,
	}
}

// defaultConfig default config
func defaultConfig() *config {
	coreConfig := defaultCoreConfig()
	return &config{
		coreConfig: *coreConfig,
		traceConfig: &traceConfig{
			recordStackTraceInSpan: true,
			errorSpanLevel:         zapcore.ErrorLevel,
		},
		zapOpts: []zap.Option{},
	}
}

// WithCoreEnc zapcore encoder
func WithCoreEnc(enc zapcore.Encoder) Option {
	return option(func(cfg *config) {
		cfg.coreConfig.enc = enc
	})
}

// WithCoreWs zapcore write syncer
func WithCoreWs(ws zapcore.WriteSyncer) Option {
	return option(func(cfg *config) {
		cfg.coreConfig.ws = ws
	})
}

// WithCoreLevel zapcore log level
func WithCoreLevel(lvl zap.AtomicLevel) Option {
	return option(func(cfg *config) {
		cfg.coreConfig.lvl = lvl
	})
}

// WithZapOptions add origin zap option
func WithZapOptions(opts ...zap.Option) Option {
	return option(func(cfg *config) {
		cfg.zapOpts = append(cfg.zapOpts, opts...)
	})
}

// WithTraceErrorSpanLevel trace error span level option
func WithTraceErrorSpanLevel(level zapcore.Level) Option {
	return option(func(cfg *config) {
		cfg.traceConfig.errorSpanLevel = level
	})
}

// WithRecordStackTraceInSpan record stack track option
func WithRecordStackTraceInSpan(recordStackTraceInSpan bool) Option {
	return option(func(cfg *config) {
		cfg.traceConfig.recordStackTraceInSpan = recordStackTraceInSpan
	})
}
