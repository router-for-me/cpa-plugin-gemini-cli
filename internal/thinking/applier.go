package thinking

import (
	"context"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type Applier struct{}

func NewApplier() *Applier { return &Applier{} }

func (a *Applier) Identifier() string { return "gemini-cli" }

func (a *Applier) ApplyThinking(_ context.Context, req pluginapi.ThinkingApplyRequest) (pluginapi.PayloadResponse, error) {
	body := append([]byte(nil), req.Body...)
	config := normalizeThinkingConfig(req.Config)
	path := "request.generationConfig.thinkingConfig"
	switch config.Mode {
	case "none":
		updated, errDelete := sjson.DeleteBytes(body, path)
		if errDelete == nil {
			body = updated
		}
	case "budget":
		body = setThinkingBool(body, path+".includeThoughts", true)
		body = setThinkingInt(body, path+".thinkingBudget", config.Budget)
	case "level":
		body = setThinkingBool(body, path+".includeThoughts", true)
		body = setThinkingString(body, path+".thinkingLevel", config.Level)
	case "auto":
		body = setThinkingBool(body, path+".includeThoughts", true)
		if gjson.GetBytes(body, path+".thinkingBudget").Exists() {
			updated, errDelete := sjson.DeleteBytes(body, path+".thinkingBudget")
			if errDelete == nil {
				body = updated
			}
		}
	}
	return pluginapi.PayloadResponse{Body: body}, nil
}

func normalizeThinkingConfig(config pluginapi.ThinkingConfig) pluginapi.ThinkingConfig {
	config.Mode = strings.ToLower(strings.TrimSpace(config.Mode))
	config.Level = strings.ToLower(strings.TrimSpace(config.Level))
	switch config.Mode {
	case "none", "budget", "level", "auto":
	default:
		config.Mode = "auto"
	}
	if config.Mode == "level" && config.Level == "" {
		config.Mode = "auto"
	}
	if config.Mode == "budget" && config.Budget <= 0 {
		config.Mode = "none"
	}
	return config
}

func setThinkingBool(body []byte, path string, value bool) []byte {
	updated, errSet := sjson.SetBytes(body, path, value)
	if errSet != nil {
		return body
	}
	return updated
}

func setThinkingInt(body []byte, path string, value int) []byte {
	updated, errSet := sjson.SetBytes(body, path, value)
	if errSet != nil {
		return body
	}
	return updated
}

func setThinkingString(body []byte, path string, value string) []byte {
	updated, errSet := sjson.SetBytes(body, path, value)
	if errSet != nil {
		return body
	}
	return updated
}
