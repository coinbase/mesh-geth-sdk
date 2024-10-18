// Copyright 2024 Coinbase, Inc.
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

package client

import "net/http"

func NewDefaultHTTPTransport() http.RoundTripper {
	// Override transport idle connection settings
	//
	// See this conversation around why `.Clone()` is used here:
	// https://github.com/golang/go/issues/26013
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.IdleConnTimeout = DefaultIdleConnTimeout
	defaultTransport.MaxIdleConns = DefaultMaxConnections
	defaultTransport.MaxIdleConnsPerHost = DefaultMaxConnections

	return defaultTransport
}
