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
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

type Timer struct {
	statsdClient *statsd.Client
	name         string
	tags         map[string]string
	startTime    time.Time
}

func NewTimer(s *statsd.Client, name string, tags ...string) *Timer {
	return &Timer{
		statsdClient: s,
		name:         name,
		tags:         getTags(tags...),
		startTime:    time.Now(),
	}
}

func (t *Timer) Emit() {
	t.statsdClient.Timing(t.name, time.Since(t.startTime), makeTagsSlice(baseTags, t.tags), 1)
}

func Incr(s *statsd.Client, name string, tags ...string) {
	s.Incr(name, makeTagsSlice(baseTags, getTags(tags...)), 1)
}

// Equivalent to time.Duration.Milliseconds except uses float division
// instead of integer division so it's less lossy
func FloatMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func getTags(options ...string) map[string]string {
	tags := make(map[string]string)

	if len(options)%2 == 0 {
		for i := 1; i < len(options); i += 2 {
			tags[options[i-1]] = options[i]
		}
	}

	return tags
}
