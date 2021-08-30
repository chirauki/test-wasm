// Copyright 2020-2021 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"test-wasm/pkg/common"

	"github.com/buger/jsonparser"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	tickMilliseconds uint32 = 1000 * 10
	randomDataHost          = "random-data-api.com"
	randomDataPath          = "/api/beer/random_beer"
	// randomDataClusterName        = "random_data"
	randomDataClusterName = "outbound|443||random-data-api.com"
)

func main() {
	proxywasm.SetVMContext(&vmContext{})
}

type vmContext struct {
	// Embed the default VM context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultVMContext
}

// Override types.DefaultVMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{}
}

type pluginContext struct {
	// Embed the default plugin context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultPluginContext
	contextID uint32
	callNum   uint32
	callBack  func(numHeaders, bodySize, numTrailers int)
}

func (ctx *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	if err := proxywasm.SetTickPeriodMilliSeconds(tickMilliseconds); err != nil {
		proxywasm.LogCriticalf("failed to set tick period: %v", err)
		return types.OnPluginStartStatusFailed
	}
	proxywasm.LogInfof("set tick period milliseconds: %d", tickMilliseconds)
	proxywasm.SetSharedData(common.SharedKeyData, []byte("0"), 0)
	ctx.callBack = func(numHeaders, bodySize, numTrailers int) {
		body, err := proxywasm.GetHttpCallResponseBody(0, 65535)
		if err != nil {
			proxywasm.LogErrorf("error getting random data respose body: %v", err)
		}
		proxywasm.LogInfof("random data: %s", body)
		beerName, err := jsonparser.GetString(body, "name")
		if err != nil {
			proxywasm.LogErrorf("error getting beer name: %v", err)
		}
		err = ctx.setRandomData([]byte(beerName))
		if err != nil {
			proxywasm.LogErrorf("error storing data: %v", err)
		}
	}
	return types.OnPluginStartStatusOK
}

// Override types.DefaultPluginContext.
func (ctx *pluginContext) OnTick() {
	hs := [][2]string{
		{":method", "GET"},
		{":authority", randomDataHost},
		{":path", randomDataPath},
		{"accept", "*/*"},
	}
	if _, err := proxywasm.DispatchHttpCall(randomDataClusterName, hs, nil, nil, 5000, ctx.callBack); err != nil {
		proxywasm.LogCriticalf("dispatch httpcall failed: %v", err)
	}
}

func (ctx *pluginContext) setRandomData(data []byte) error {
	val, cas, err := proxywasm.GetSharedData(common.SharedKeyData)
	if err != nil {
		return err
	}
	proxywasm.LogInfof("data changed from %s to %s", val, data)
	return proxywasm.SetSharedData(common.SharedKeyData, []byte(data), cas)
}
