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
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"

	"github.com/DataDog/datadog-go/statsd"
	"go.uber.org/zap"

	"github.com/coinbase/rosetta-geth-sdk/configuration"

	"github.com/coinbase/rosetta-sdk-go/types"
)

const (

	// Metrics constants
	metricBlockchainClientTiming = "rosetta_api.client.timing"
	tagBlockchainClientMethod    = "client.method"
	tagRosettaAPIErrorType       = "errortype"
	metricRosettaAPIError        = "rosetta_api.client.error"

	// Logging Constants
	errorCodeKey   = "rosetta_error_code"
	apiEndpointKey = "api_endpoint"

	// API Endpoint Method
	AccountBalanceKey         = "/account/balance"
	AccountCoinsKey           = "/account/coins"
	BlockKey                  = "/block"
	BlockTransactionKey       = "/block/transaction"
	CallKey                   = "/call"
	ConstructionDeriveKey     = "/construction/derive"
	ConstructionPreprocessKey = "/construction/preprocess"
	ConstructionMetadataKey   = "/construction/metadata"
	ConstructionPayloadsKey   = "/construction/payloads"
	ConstructionCombineKey    = "/construction/combine"
	ConstructionHashKey       = "/construction/hash"
	ConstructionParseKey      = "/construction/parse"
	ConstructionSubmitKey     = "/construction/submit"
	MempoolKey                = "/mempool"
	MempoolTransactionKey     = "/mempool/transaction"
	NetworkListKey            = "/network/list"
	NetworkOptionsKey         = "/network/options"
	NetworkStatusKey          = "/network/status"
)

var baseTags []string

// InitLogger initializes and returns a zap logger and returns a function to
// sync the logs and flush the buffer.
func InitLogger(cfg *configuration.Configuration, fields ...zap.Field) (*zap.Logger, func(), error) {
	// Set up logger.
	log, err := zap.NewProduction()
	if err != nil {
		return nil, nil, err
	}

	// Configure sync function
	syncFn := func() {
		if logErr := log.Sync(); logErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to sync log %v\n", logErr)
		}
	}

	return log, syncFn, err
}

// InitStatsd configures the statsd client, starts tracing, and starts profiling.
func InitStatsd(log *zap.Logger, cfg *configuration.Configuration) (*statsd.Client, func(), error) {
	// add default tags to all metrics
	serviceTags := map[string]string{
		"blockchain": cfg.Network.Blockchain,
		"service":    cfg.ServiceName,
		"network":    cfg.Network.Network,
	}
	defaultTags := generateDefaultTagsMap()
	baseTagsMap := mergeTagsMap(serviceTags, defaultTags)
	baseTags = makeTagsSlice([]string{}, baseTagsMap)

	// Set up Statsd.
	statsdClient, err := statsd.New(cfg.StatsdAddress)
	if err != nil {
		return nil, nil, err
	}

	// Configure tracer
	tracer.Start(
		tracer.WithAgentAddr(cfg.StatsdTraceAddress),
		tracer.WithServiceName(cfg.ServiceName),
		tracer.WithGlobalTag("blockchain", cfg.Network.Blockchain),
		tracer.WithGlobalTag("network", cfg.Network.Network),
	)

	// Configure profiler
	err = profiler.Start(
		profiler.WithService(cfg.ServiceName),
		profiler.WithTags(
			"blockchain:"+cfg.Network.Blockchain,
			"network:"+cfg.Network.Network,
			"servicename:"+cfg.ServiceName,
		),
	)
	if err != nil {
		tracer.Stop()
		log.Fatal("Failed to start profiler", zap.Error(err))
	}

	done := func() {
		tracer.Stop()
		profiler.Stop()
	}

	return statsdClient, done, nil
}

func InitBlockchainClientTimer(s *statsd.Client, callMethod string) *Timer {
	return NewTimer(s, metricBlockchainClientTiming, tagBlockchainClientMethod, callMethod)
}

func IncrementErrorCount(s *statsd.Client, callMethod string, errorType string) {
	Incr(s, metricRosettaAPIError, tagBlockchainClientMethod, callMethod, tagRosettaAPIErrorType, errorType)
}

func LogError(l *zap.Logger, msg string, callMethod string, err *types.Error) {
	fields := getFields(apiEndpointKey, callMethod, errorCodeKey, strconv.FormatInt(int64(err.Code), 10))
	l.Error(msg, fields...)
}

func getFields(options ...string) []zap.Field {
	var tags []zap.Field

	if len(options)%2 == 0 {
		for i := 1; i < len(options); i += 2 {
			tags = append(tags, zap.String(options[i-1], options[i]))
		}
	}

	return tags
}

func generateDefaultTagsMap() map[string]string {
	defaultTagsMap := make(map[string]string)

	defaultTagsMap["projectname"] = os.Getenv("CODEFLOW_PROJECT_NAME")
	defaultTagsMap["configname"] = os.Getenv("CODEFLOW_CONFIG_NAME")
	defaultTagsMap["servicename"] = os.Getenv("CODEFLOW_SERVICE_NAME")
	defaultTagsMap["environment"] = os.Getenv("CODEFLOW_ENVIRONMENT")
	defaultTagsMap["env"] = os.Getenv("CODEFLOW_ENVIRONMENT")
	defaultTagsMap["releaseid"] = os.Getenv("CODEFLOW_RELEASE_ID")

	const nameVar = "METADATA_CONTAINER_NAME"
	containerName := os.Getenv(nameVar)
	if containerName == "" {
		return defaultTagsMap
	}
	defaultTagsMap["container_name"] = containerName
	return defaultTagsMap
}

func mergeTagsMap(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, mp := range maps {
		for k, v := range mp {
			result[k] = v
		}
	}
	return result
}

// baseTags: slice of strings formatted as "key:value"
// tagsMap: map containing other tags
// The output is a slice that contains baseTags, plus the items in tagsMap converted as "key:value" strings
func makeTagsSlice(baseTags []string, tagsMap map[string]string) []string {
	if len(tagsMap) == 0 {
		return baseTags
	}
	result := make([]string, len(baseTags), len(baseTags)+len(tagsMap))
	copy(result, baseTags)
	var sb strings.Builder
	for k, v := range tagsMap {
		sb.Grow(len(k) + len(v) + 1)
		sb.WriteString(k)
		sb.WriteString(":")
		sb.WriteString(v)
		result = append(result, sb.String())
		sb.Reset()
	}
	return result
}
