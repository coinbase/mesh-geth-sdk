// Copyright 2023 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stats

import (
	"bytes"
	"fmt"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"go.uber.org/zap"
)

// Logger implements the Statsd interface but logs the output
type Logger struct {
	log *zap.Logger
}

// NewLogger returns Logger
func NewLogger(log *zap.Logger) *Logger {
	if log == nil {
		log = zap.NewNop()
	}
	return &Logger{
		log: log,
	}
}

// GaugeWithTags logs a gauge metric
func (l *Logger) GaugeWithTags(name string, value float64, tags map[string]string) {
	l.log.Info("stat: gauge",
		zap.String("key", name),
		zap.Float64("val_float64", value),
		zap.String("tags", tagsToString(tags)),
	)
}

// Gauge logs a gauge metric
func (l *Logger) Gauge(name string, value float64) {
	l.log.Info("stat: gauge",
		zap.String("key", name),
		zap.Float64("val_float64", value),
	)
}

// Histogram logs a histogram metric
func (l *Logger) Histogram(name string, value float64) {
	l.log.Info("stat: histogram",
		zap.String("key", name),
		zap.Float64("val_float64", value),
	)
}

// Distribution logs a distribution metric
func (l *Logger) Distribution(name string, value float64) {
	l.log.Info("stat: distribution",
		zap.String("key", name),
		zap.Float64("val_float64", value),
	)
}

// Incr logs a count metric with an increment of 1
func (l *Logger) Incr(name string) {
	l.log.Info("stat: increment", zap.String("key", name))
}

// Event logs sending an event
func (l *Logger) Event(_ *statsd.Event) {
	// To Do: Supplement this with more information
	l.log.Info("event")
}

// TimingWithTags logs a timing metric with tags
func (l *Logger) TimingWithTags(name string, value time.Duration, tags map[string]string) {
	l.log.Info("stat: timing",
		zap.String("key", name),
		zap.Float64("val_float64", float64(value.Milliseconds())),
		zap.String("tags", tagsToString(tags)),
	)
}

// DistributionWithTags logs a distribution metric with tags
func (l *Logger) DistributionWithTags(name string, value float64, tags map[string]string) {
	l.log.Info("stat: distribution",
		zap.String("key", name),
		zap.Float64("val_float64", value),
		zap.String("tags", tagsToString(tags)),
	)
}

// tagsToString converts tags into a string representation
func tagsToString(tags map[string]string) string {
	buffer := bytes.NewBufferString("")
	for k, v := range tags {
		// https://godoc.org/bytes#Buffer.WriteString
		// The return value n is the length of s; err is always nil.
		_, err := buffer.WriteString(fmt.Sprintf("%s=%s,", k, v))
		if err != nil {
			fmt.Print(err)
			//_ = bugsnag.Notify(err)
		}
	}
	return buffer.String()
}
