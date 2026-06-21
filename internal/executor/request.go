package executor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	authpkg "github.com/router-for-me/CLIProxyAPIPlugins/geminicli/internal/auth"
	"github.com/router-for-me/CLIProxyAPIPlugins/geminicli/internal/compat"
	"github.com/router-for-me/CLIProxyAPIPlugins/geminicli/internal/fingerprint"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	baseURL             = "https://cloudcode-pa.googleapis.com"
	generatePath        = "/v1internal:generateContent"
	streamPath          = "/v1internal:streamGenerateContent"
	countTokensPath     = "/v1internal:countTokens"
	defaultErrorMessage = "gemini-cli upstream request failed"
)

type BuiltRequest struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
}

func BuildRequestInput(storageJSON []byte, metadata map[string]any, attributes map[string]string, model string, payload []byte, action string, alt string) (BuiltRequest, error) {
	storage, token, errToken := storageAndAccessToken(storageJSON)
	if errToken != nil {
		return BuiltRequest{}, errToken
	}
	projectID := projectIDFromAuth(metadata, attributes, storage)
	body := geminiCLIRequestBody(model, payload)
	if action == "countTokens" {
		body = deletePath(body, "project")
		body = deletePath(body, "model")
	} else {
		if strings.TrimSpace(projectID) == "" {
			return BuiltRequest{}, fmt.Errorf("gemini-cli project_id is required")
		}
		body = setString(body, "project", projectID)
		body = setString(body, "model", model)
	}
	path := generatePath
	if action == "streamGenerateContent" {
		path = streamPath
	} else if action == "countTokens" {
		path = countTokensPath
	}
	endpoint := baseURL + path
	if action == "streamGenerateContent" {
		if strings.TrimSpace(alt) == "" {
			endpoint += "?alt=sse"
		} else {
			endpoint += "?alt=" + strings.TrimSpace(alt)
		}
	} else if strings.TrimSpace(alt) != "" {
		endpoint += "?alt=" + strings.TrimSpace(alt)
	}
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", acceptHeader(action))
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("User-Agent", fingerprint.UserAgent(model))
	headers.Set("X-Goog-Api-Client", fingerprint.APIClientHeader)
	applyCustomHeaders(headers, metadata, attributes)
	return BuiltRequest{Method: http.MethodPost, URL: endpoint, Headers: headers, Body: body}, nil
}

func geminiCLIRequestBody(model string, payload []byte) []byte {
	if gjson.GetBytes(payload, "request").Exists() {
		return append([]byte(nil), payload...)
	}
	return compat.WrapRequest(model, payload)
}

func accessTokenFromStorage(storageJSON []byte) (string, error) {
	_, token, errToken := storageAndAccessToken(storageJSON)
	return token, errToken
}

func storageAndAccessToken(storageJSON []byte) (authpkg.Storage, string, error) {
	storage, errParse := authpkg.ParseStorage(storageJSON)
	if errParse != nil {
		return authpkg.Storage{}, "", errParse
	}
	if storage == nil {
		return authpkg.Storage{}, "", fmt.Errorf("gemini-cli auth storage is missing")
	}
	token := strings.TrimSpace(storage.AccessToken)
	if token == "" {
		token = strings.TrimSpace(gjson.GetBytes(storageJSON, "token.access_token").String())
	}
	if token == "" {
		return authpkg.Storage{}, "", fmt.Errorf("gemini-cli access token is missing")
	}
	return *storage, token, nil
}

func projectIDFromAuth(metadata map[string]any, attributes map[string]string, storage authpkg.Storage) string {
	if attributes != nil {
		if projectID := strings.TrimSpace(attributes["project_id"]); projectID != "" {
			return projectID
		}
	}
	if metadata != nil {
		if projectID, ok := metadata["project_id"].(string); ok && strings.TrimSpace(projectID) != "" {
			return strings.TrimSpace(projectID)
		}
	}
	return strings.TrimSpace(storage.ProjectID)
}

func acceptHeader(action string) string {
	if action == "streamGenerateContent" {
		return "text/event-stream"
	}
	return "application/json"
}

func applyCustomHeaders(headers http.Header, metadata map[string]any, attributes map[string]string) {
	for key, value := range attributes {
		if !strings.HasPrefix(key, "header:") {
			continue
		}
		headerName := strings.TrimSpace(strings.TrimPrefix(key, "header:"))
		if headerName != "" && strings.TrimSpace(value) != "" {
			headers.Set(headerName, strings.TrimSpace(value))
		}
	}
	rawHeaders, ok := metadata["headers"]
	if !ok || rawHeaders == nil {
		return
	}
	raw, errMarshal := json.Marshal(rawHeaders)
	if errMarshal != nil {
		return
	}
	var decoded map[string]string
	if errUnmarshal := json.Unmarshal(raw, &decoded); errUnmarshal != nil {
		return
	}
	for key, value := range decoded {
		headerName := strings.TrimSpace(key)
		if headerName != "" && strings.TrimSpace(value) != "" {
			headers.Set(headerName, strings.TrimSpace(value))
		}
	}
}

func setString(body []byte, path string, value string) []byte {
	updated, errSet := sjson.SetBytes(body, path, value)
	if errSet != nil {
		return body
	}
	return updated
}

func deletePath(body []byte, path string) []byte {
	updated, errDelete := sjson.DeleteBytes(body, path)
	if errDelete != nil {
		return body
	}
	return updated
}
