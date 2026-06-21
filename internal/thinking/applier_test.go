package thinking

import (
	"context"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"github.com/tidwall/gjson"
)

func TestApplyThinkingNoneDeletesThinkingConfig(t *testing.T) {
	body := []byte(`{"request":{"generationConfig":{"thinkingConfig":{"includeThoughts":true,"thinkingBudget":1024},"temperature":0.3}}}`)

	resp, errApply := NewApplier().ApplyThinking(context.Background(), pluginapi.ThinkingApplyRequest{
		Config: pluginapi.ThinkingConfig{Mode: "none"},
		Body:   body,
	})
	if errApply != nil {
		t.Fatalf("ApplyThinking returned error: %v", errApply)
	}
	if gjson.GetBytes(resp.Body, "request.generationConfig.thinkingConfig").Exists() {
		t.Fatalf("thinkingConfig was not deleted: %s", resp.Body)
	}
	if got := gjson.GetBytes(resp.Body, "request.generationConfig.temperature").Float(); got != 0.3 {
		t.Fatalf("temperature = %v, want 0.3", got)
	}
}

func TestApplyThinkingBudgetSetsBudget(t *testing.T) {
	resp, errApply := NewApplier().ApplyThinking(context.Background(), pluginapi.ThinkingApplyRequest{
		Config: pluginapi.ThinkingConfig{Mode: "budget", Budget: 4096},
		Body:   []byte(`{"request":{"generationConfig":{}}}`),
	})
	if errApply != nil {
		t.Fatalf("ApplyThinking returned error: %v", errApply)
	}
	if !gjson.GetBytes(resp.Body, "request.generationConfig.thinkingConfig.includeThoughts").Bool() {
		t.Fatalf("includeThoughts was not enabled: %s", resp.Body)
	}
	if got := gjson.GetBytes(resp.Body, "request.generationConfig.thinkingConfig.thinkingBudget").Int(); got != 4096 {
		t.Fatalf("thinkingBudget = %d, want 4096", got)
	}
}

func TestApplyThinkingLevelSetsLevel(t *testing.T) {
	resp, errApply := NewApplier().ApplyThinking(context.Background(), pluginapi.ThinkingApplyRequest{
		Config: pluginapi.ThinkingConfig{Mode: "level", Level: "HIGH"},
		Body:   []byte(`{"request":{"generationConfig":{}}}`),
	})
	if errApply != nil {
		t.Fatalf("ApplyThinking returned error: %v", errApply)
	}
	if got := gjson.GetBytes(resp.Body, "request.generationConfig.thinkingConfig.thinkingLevel").String(); got != "high" {
		t.Fatalf("thinkingLevel = %q, want high", got)
	}
}
