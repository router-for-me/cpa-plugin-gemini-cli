package main

import (
	"encoding/json"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
)

func TestHandleRegisterDeclaresCapabilities(t *testing.T) {
	raw, errRegister := handleRegister([]byte(`{}`))
	if errRegister != nil {
		t.Fatalf("handleRegister returned error: %v", errRegister)
	}
	defer GeminiCLIPluginShutdown()

	var envelope pluginabi.Envelope
	if errDecode := json.Unmarshal(raw, &envelope); errDecode != nil {
		t.Fatalf("decode envelope: %v", errDecode)
	}
	if !envelope.OK {
		t.Fatalf("registration envelope is not OK: %#v", envelope.Error)
	}
	var registration abiRegistration
	if errDecode := json.Unmarshal(envelope.Result, &registration); errDecode != nil {
		t.Fatalf("decode registration: %v", errDecode)
	}
	if registration.SchemaVersion != pluginabi.SchemaVersion {
		t.Fatalf("schema version = %d, want %d", registration.SchemaVersion, pluginabi.SchemaVersion)
	}
	if registration.Metadata.Version != pluginVersion {
		t.Fatalf("metadata version = %q, want %q", registration.Metadata.Version, pluginVersion)
	}
	if !registration.Capabilities.AuthProvider || !registration.Capabilities.Executor {
		t.Fatalf("missing auth or executor capability: %#v", registration.Capabilities)
	}
	if !registration.Capabilities.RequestTranslator || !registration.Capabilities.ResponseTranslator {
		t.Fatalf("missing translator capability: %#v", registration.Capabilities)
	}
	if len(registration.Capabilities.ExecutorInputFormats) != 1 || registration.Capabilities.ExecutorInputFormats[0] != "gemini" {
		t.Fatalf("executor input formats = %#v", registration.Capabilities.ExecutorInputFormats)
	}
	if len(registration.Capabilities.ExecutorOutputFormats) != 1 || registration.Capabilities.ExecutorOutputFormats[0] != "gemini" {
		t.Fatalf("executor output formats = %#v", registration.Capabilities.ExecutorOutputFormats)
	}
}
