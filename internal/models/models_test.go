package models

import (
	"context"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestStaticModelsIncludeExpectedGeminiCLIMetadata(t *testing.T) {
	resp, errModels := NewProvider().StaticModels(context.Background(), pluginapi.StaticModelRequest{})
	if errModels != nil {
		t.Fatalf("StaticModels returned error: %v", errModels)
	}
	if len(resp.Models) == 0 {
		t.Fatal("StaticModels returned no models")
	}
	byID := make(map[string]pluginapi.ModelInfo, len(resp.Models))
	for _, model := range resp.Models {
		byID[model.ID] = model
	}
	model, ok := byID["gemini-2.5-pro"]
	if !ok {
		t.Fatalf("gemini-2.5-pro missing from models: %#v", byID)
	}
	if model.Thinking == nil {
		t.Fatal("gemini-2.5-pro thinking support is nil")
	}
	if model.Thinking.Max != 65536 {
		t.Fatalf("thinking max = %d, want 65536", model.Thinking.Max)
	}
	if !containsLevel(model.Thinking.Levels, "xhigh") {
		t.Fatalf("thinking levels = %#v, want xhigh", model.Thinking.Levels)
	}
}

func containsLevel(levels []string, want string) bool {
	for _, level := range levels {
		if level == want {
			return true
		}
	}
	return false
}
