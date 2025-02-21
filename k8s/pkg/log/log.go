/** Copyright 2020-2023 Alibaba Group Holding Limited.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package log contains the global logger for the vineyard operator.
package log

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"

	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	defaultLogger = zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}))
	dlog = log.NewDelegatingLogSink(defaultLogger.GetSink())

	Log = Logger{logr.New(dlog).WithName("vineyard")}
)

// Extends the logr's logger with Fatal support
type Logger struct {
	logr.Logger
}

// SetLogger sets a concrete logging implementation for all deferred Loggers.
func SetLogger(l Logger) {
	dlog.Fulfill(l.GetSink())
}

// FromContext returns a logger with predefined values from a context.Context.
func FromContext(ctx context.Context, keysAndValues ...any) Logger {
	log := Log.Logger
	if ctx != nil {
		if logger, err := logr.FromContext(ctx); err == nil {
			log = logger
		}
	}
	return Logger{log.WithValues(keysAndValues...)}
}

// IntoContext takes a context and sets the logger as one of its values.
// Use FromContext function to retrieve the logger.
func IntoContext(ctx context.Context, log Logger) context.Context {
	return logr.NewContext(ctx, log.Logger)
}

func V(level int) Logger {
	return Logger{Log.V(level)}
}

func WithValues(keysAndValues ...any) Logger {
	return Logger{Log.WithValues(keysAndValues...)}
}

func WithName(name string) Logger {
	return Logger{Log.WithName(name)}
}

func (l Logger) Fatal(err error, msg string, keysAndValues ...any) {
	l.Error(err, msg, keysAndValues...)
	os.Exit(1)
}

func (l Logger) Infof(format string, v ...any) {
	l.Info(fmt.Sprintf(format, v...))
}

func (l Logger) Errorf(err error, format string, v ...any) {
	l.Error(err, fmt.Sprintf(format, v...))
}

func (l Logger) Fatalf(err error, format string, v ...any) {
	l.Fatal(err, fmt.Sprintf(format, v...))
}

func (l Logger) Output(msg string) {
	fmt.Println(msg)
}

func (l Logger) Outputf(format string, v ...any) {
	fmt.Printf(format, v...)
}

func Info(msg string, keysAndValues ...any) {
	Log.Info(msg, keysAndValues...)
}

func Error(err error, msg string, keysAndValues ...any) {
	Log.Error(err, msg, keysAndValues...)
}

func Fatal(err error, msg string, keysAndValues ...any) {
	Log.Fatal(err, msg, keysAndValues...)
}

func Infof(format string, v ...any) {
	Log.Infof(format, v...)
}

func Errorf(err error, format string, v ...any) {
	Log.Errorf(err, format, v...)
}

func Fatalf(err error, format string, v ...any) {
	Log.Fatalf(err, format, v...)
}

func Output(msg string) {
	Log.Output(msg)
}

func Outputf(msg string, keysAndValues ...any) {
	Log.Outputf(msg, keysAndValues...)
}
