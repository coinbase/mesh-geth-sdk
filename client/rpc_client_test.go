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

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newNode answers every JSON-RPC request with 0x1.
func newNode(t *testing.T) (url string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x1"}`))
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

// recordingTransport records the JSON-RPC method of every request sent through it.
type recordingTransport struct {
	mu      sync.Mutex
	methods []string
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	var call struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(body, &call)

	rt.mu.Lock()
	rt.methods = append(rt.methods, call.Method)
	rt.mu.Unlock()

	return http.DefaultTransport.RoundTrip(req)
}

func TestNewRPCClient(t *testing.T) {
	transport := &recordingTransport{}

	c, err := NewRPCClient(newNode(t), transport)
	require.NoError(t, err)

	var result string
	require.NoError(t, c.CallContext(context.Background(), &result, "eth_blockNumber"))

	assert.Equal(t, "0x1", result)
	assert.Equal(t, []string{"eth_blockNumber"}, transport.methods)
}
