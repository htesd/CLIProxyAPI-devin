package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	void* call;
	void* free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);
*/
import "C"

import (
	"encoding/json"
	"strings"
	"unsafe"

	"github.com/MoeForge/moego/ojson"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

// devin-nofast strips fast/priority selection from requests after they are
// translated to the Devin (interactions) format and before they go upstream:
//   - service_tier is removed (Codex /fast sends "priority");
//   - thinking_level "fast"/"priority" is removed so the model default applies;
//   - thinking_level "<level>-fast" / "<level>-priority" keeps only "<level>".
// A "(fast)" model-name suffix cannot be rewritten by a request normalizer and
// must be blocked by the client-facing model allowlist.

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type registration struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Metadata      pluginapi.Metadata     `json:"metadata"`
	Capabilities  registrationCapability `json:"capabilities"`
}

type registrationCapability struct {
	RequestNormalizer bool `json:"request_normalizer"`
}

var serviceTierPaths = []string{"service_tier", "serviceTier", "generation_config.service_tier", "generationConfig.serviceTier"}
var levelPaths = []string{"generation_config.thinking_level", "generationConfig.thinking_level", "generationConfig.thinkingLevel", "reasoning_effort", "reasoning.effort"}
var fastSuffixes = []string{"-fast", "-priority"}

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(_ *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	plugin.abi_version = C.uint32_t(pluginabi.ABIVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}
	var requestBytes []byte
	if request != nil && requestLen > 0 {
		requestBytes = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	raw, errHandle := handleMethod(C.GoString(method), requestBytes)
	if errHandle != nil {
		writeResponse(response, errorEnvelope("plugin_error", errHandle.Error()))
		return 1
	}
	writeResponse(response, raw)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, len C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		return okEnvelope(pluginRegistration())
	case pluginabi.MethodRequestNormalize:
		return normalizeRequest(request)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             "devin-nofast",
			Version:          "0.1.0",
			Author:           "iiap",
			GitHubRepository: "https://github.com/router-for-me/CLIProxyAPI",
		},
		Capabilities: registrationCapability{RequestNormalizer: true},
	}
}

func normalizeRequest(raw []byte) ([]byte, error) {
	var req pluginapi.RequestTransformRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	if !strings.EqualFold(req.ToFormat, "interactions") {
		return okEnvelope(pluginapi.PayloadResponse{Body: req.Body})
	}
	return okEnvelope(pluginapi.PayloadResponse{Body: stripFast(req.Body)})
}

// stripFast returns body without fast/priority selection. Any failed edit keeps
// the previous value so the plugin never corrupts a request.
func stripFast(body []byte) []byte {
	out := body
	for _, path := range serviceTierPaths {
		if !ojson.GetBytes(out, path).Exists() {
			continue
		}
		if updated, errDel := ojson.DeleteBytes(out, path); errDel == nil {
			out = updated
		}
	}
	for _, path := range levelPaths {
		res := ojson.GetBytes(out, path)
		if !res.Exists() || res.Type != ojson.String {
			continue
		}
		level := strings.ToLower(strings.TrimSpace(res.String()))
		if level == "fast" || level == "priority" {
			if updated, errDel := ojson.DeleteBytes(out, path); errDel == nil {
				out = updated
			}
			continue
		}
		for _, suffix := range fastSuffixes {
			if strings.HasSuffix(level, suffix) {
				if updated, errSet := ojson.SetBytes(out, path, strings.TrimSuffix(level, suffix)); errSet == nil {
					out = updated
				}
				break
			}
		}
	}
	return out
}

func okEnvelope(v any) ([]byte, error) {
	raw, errMarshal := json.Marshal(v)
	if errMarshal != nil {
		return nil, errMarshal
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}
