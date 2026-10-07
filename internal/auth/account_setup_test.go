package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type accountSetupClient struct {
	load           string
	onboard        string
	loadStatus     int
	loadProject    string
	onboardProject string
	projectReads   int
	onboardCalls   int
	operationReads int
	cancelOnboard  context.CancelFunc
}

func (c *accountSetupClient) Do(_ context.Context, req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
	respond := func(status int, body string) (pluginapi.HTTPResponse, error) {
		return pluginapi.HTTPResponse{StatusCode: status, Body: []byte(body)}, nil
	}
	switch req.URL {
	case tokenURL:
		return respond(http.StatusOK, `{"access_token":"fixture-access","refresh_token":"fixture-refresh","token_type":"Bearer"}`)
	case userInfoURL:
		return respond(http.StatusOK, `{"email":"fixture@example.com"}`)
	case "https://cloudresourcemanager.googleapis.com/v1/projects":
		c.projectReads++
		return respond(http.StatusOK, `{"projects":[{"projectId":"unrelated-unlicensed"},{"projectId":"another-unrelated"}]}`)
	case codeAssistBaseURL + "/" + codeAssistVersion + ":loadCodeAssist":
		var body map[string]any
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return pluginapi.HTTPResponse{}, err
		}
		if project, ok := body["cloudaicompanionProject"].(string); ok {
			c.loadProject = project
		}
		if c.loadProject == "unrelated-unlicensed" {
			return respond(http.StatusForbidden, `{"error":{"status":"PERMISSION_DENIED","details":[{"reason":"SUBSCRIPTION_REQUIRED"}]}}`)
		}
		status := c.loadStatus
		if status == 0 {
			status = http.StatusOK
		}
		return respond(status, c.load)
	case codeAssistBaseURL + "/" + codeAssistVersion + ":onboardUser":
		c.onboardCalls++
		if c.cancelOnboard != nil {
			c.cancelOnboard()
		}
		var body map[string]any
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return pluginapi.HTTPResponse{}, err
		}
		if project, ok := body["cloudaicompanionProject"].(string); ok {
			c.onboardProject = project
		}
		if c.onboard == "" {
			return pluginapi.HTTPResponse{}, fmt.Errorf("unexpected onboarding of an existing account")
		}
		return respond(http.StatusOK, c.onboard)
	case codeAssistBaseURL + "/" + codeAssistVersion + "/operations/fixture":
		if req.Method != http.MethodGet || len(req.Body) != 0 {
			return pluginapi.HTTPResponse{}, fmt.Errorf("operation polling must use GET without a body")
		}
		c.operationReads++
		if c.operationReads == 1 {
			return respond(http.StatusOK, `{"done":false}`)
		}
		return respond(http.StatusOK, `{"done":true,"response":{"cloudaicompanionProject":{"id":"managed-personal"}}}`)
	default:
		// Preview-channel setup is best effort and outside the login contract.
		return respond(http.StatusNotFound, `{}`)
	}
}

func (c *accountSetupClient) DoStream(context.Context, pluginapi.HTTPRequest) (pluginapi.HTTPStreamResponse, error) {
	return pluginapi.HTTPStreamResponse{}, fmt.Errorf("unexpected stream")
}

func TestPersonalLoginUsesAccountDiscoveryWithoutCloudProjectEnumeration(t *testing.T) {
	c := &accountSetupClient{load: `{"currentTier":{"id":"standard-tier"},"paidTier":{"id":"g1-pro-tier"},"cloudaicompanionProject":"managed-personal"}`}
	s, err := NewProvider().finalizeCode(context.Background(), c, "fixture-code", map[string]any{"redirect_uri": "http://127.0.0.1/fixture"})
	if err != nil {
		t.Fatalf("personal account login failed: %v", err)
	}
	if c.projectReads != 0 || c.loadProject != "" || c.onboardCalls != 0 {
		t.Fatalf("account discovery: cloud reads=%d, selected project=%q, onboarding=%d", c.projectReads, c.loadProject, c.onboardCalls)
	}
	if s.ProjectID != "managed-personal" || len(s.ProjectIDs) != 1 || s.ProjectIDs[0] != s.ProjectID {
		t.Fatalf("saved unvalidated projects: %#v", s.ProjectIDs)
	}
}

func TestExistingAccountUsesManagedProjectObjectWithoutOnboarding(t *testing.T) {
	c := &accountSetupClient{load: `{"currentTier":{"id":"free-tier"},"cloudaicompanionProject":{"id":"managed-personal"}}`}
	s := Storage{AccessToken: "fixture-access"}
	if err := setupCodeAssist(context.Background(), c, &s, ""); err != nil {
		t.Fatal(err)
	}
	if s.ProjectID != "managed-personal" || c.onboardCalls != 0 {
		t.Fatalf("unexpected account setup: project=%q onboarding=%d", s.ProjectID, c.onboardCalls)
	}
}

func TestFreeOnboardingOmitsProjectAndRunsOnlyOnce(t *testing.T) {
	c := &accountSetupClient{load: `{"allowedTiers":[{"id":"free-tier","isDefault":true}]}`, onboard: `{"done":true,"response":{"cloudaicompanionProject":"managed-personal"}}`}
	s := Storage{AccessToken: "fixture-access"}
	if err := setupCodeAssist(context.Background(), c, &s, "explicit-project"); err != nil {
		t.Fatal(err)
	}
	if c.loadProject != "explicit-project" || c.onboardProject != "" || c.onboardCalls != 1 || s.ProjectID != "managed-personal" {
		t.Fatalf("free onboarding: load=%q onboard=%q calls=%d result=%q", c.loadProject, c.onboardProject, c.onboardCalls, s.ProjectID)
	}
}

func TestExplicitLicensedProjectIsSavedWithoutUnrelatedProjects(t *testing.T) {
	c := &accountSetupClient{load: `{"currentTier":{"id":"standard-tier"}}`}
	s, err := NewProvider().finalizeCode(context.Background(), c, "fixture-code", map[string]any{"redirect_uri": "http://127.0.0.1/fixture", "project_id": "licensed-project"})
	if err != nil {
		t.Fatal(err)
	}
	if c.loadProject != "licensed-project" || c.projectReads != 0 || c.onboardCalls != 0 || s.ProjectID != "licensed-project" || len(s.ProjectIDs) != 1 {
		t.Fatalf("explicit project setup: %#v, reads=%d, onboarding=%d", s.ProjectIDs, c.projectReads, c.onboardCalls)
	}
}

func TestCurrentTierWithoutManagedOrExplicitProjectRefusesBeforeOnboarding(t *testing.T) {
	c := &accountSetupClient{load: `{"currentTier":{"id":"standard-tier"}}`}
	s := Storage{AccessToken: "fixture-access"}
	if err := setupCodeAssist(context.Background(), c, &s, ""); err == nil {
		t.Fatal("missing project accepted")
	}
	if c.onboardCalls != 0 || s.ProjectID != "" {
		t.Fatalf("refusal changed account setup")
	}
}

func TestOnboardingPollsReturnedOperationInsteadOfStartingAgain(t *testing.T) {
	c := &accountSetupClient{load: `{"allowedTiers":[{"id":"free-tier","isDefault":true}]}`, onboard: `{"name":"operations/fixture","done":false}`}
	s := Storage{AccessToken: "fixture-access"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := setupCodeAssist(ctx, c, &s, ""); err != nil {
		t.Fatal(err)
	}
	if c.onboardCalls != 1 || c.operationReads != 2 || s.ProjectID != "managed-personal" {
		t.Fatalf("onboarding=%d polls=%d project=%q", c.onboardCalls, c.operationReads, s.ProjectID)
	}
}

func TestProviderSubscriptionRefusalReturnsNoAuthStorage(t *testing.T) {
	c := &accountSetupClient{loadStatus: http.StatusForbidden, load: `{"error":{"status":"PERMISSION_DENIED","details":[{"reason":"SUBSCRIPTION_REQUIRED"}]}}`}
	s, err := NewProvider().finalizeCode(context.Background(), c, "fixture-code", map[string]any{"redirect_uri": "http://127.0.0.1/fixture"})
	if err == nil || !strings.Contains(err.Error(), "SUBSCRIPTION_REQUIRED") || s != nil || c.onboardCalls != 0 {
		t.Fatalf("provider refusal: storage=%v err=%v onboarding=%d", s, err, c.onboardCalls)
	}
}

func TestAccountValidationRequirementRefusesBeforeOnboarding(t *testing.T) {
	c := &accountSetupClient{load: `{"ineligibleTiers":[{"reasonCode":"VALIDATION_REQUIRED","validationUrl":"https://example.com/validate"}]}`}
	s := Storage{AccessToken: "fixture-access"}
	if err := setupCodeAssist(context.Background(), c, &s, ""); err == nil || !strings.Contains(err.Error(), "validation is required") {
		t.Fatalf("validation refusal: %v", err)
	}
	if c.onboardCalls != 0 || s.ProjectID != "" {
		t.Fatal("validation requirement changed account setup")
	}
}

func TestNonFreeOnboardingKeepsExplicitProjectWhenOperationOmitsIt(t *testing.T) {
	c := &accountSetupClient{load: `{"allowedTiers":[{"id":"standard-tier","isDefault":true}]}`, onboard: `{"done":true,"response":{}}`}
	s := Storage{AccessToken: "fixture-access"}
	if err := setupCodeAssist(context.Background(), c, &s, "licensed-project"); err != nil {
		t.Fatal(err)
	}
	if c.onboardCalls != 1 || c.onboardProject != "licensed-project" || s.ProjectID != "licensed-project" {
		t.Fatalf("explicit project was not retained: calls=%d sent=%q saved=%q", c.onboardCalls, c.onboardProject, s.ProjectID)
	}
}

func TestNonFreeAccountAcceptsServerManagedProjectFromOnboarding(t *testing.T) {
	c := &accountSetupClient{load: `{"allowedTiers":[{"id":"standard-tier","isDefault":true}]}`, onboard: `{"done":true,"response":{"cloudaicompanionProject":{"id":"managed-personal"}}}`}
	s := Storage{AccessToken: "fixture-access"}
	if err := setupCodeAssist(context.Background(), c, &s, ""); err != nil {
		t.Fatal(err)
	}
	if c.onboardCalls != 1 || c.onboardProject != "" || s.ProjectID != "managed-personal" {
		t.Fatal("account managed project was discarded")
	}
}

func TestCanceledOnboardingReturnsWithoutWaitingOrPolling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &accountSetupClient{load: `{"allowedTiers":[{"id":"free-tier","isDefault":true}]}`, onboard: `{"name":"operations/fixture","done":false}`, cancelOnboard: cancel}
	s := Storage{AccessToken: "fixture-access"}
	result := make(chan error, 1)
	go func() { result <- setupCodeAssist(ctx, c, &s, "") }()
	select {
	case err := <-result:
		if err != context.Canceled {
			t.Fatalf("cancellation: %v", err)
		}
		if c.operationReads != 0 {
			t.Fatal("canceled onboarding polled the provider")
		}
	case <-time.After(time.Second):
		t.Fatal("onboarding did not honor cancellation before its two-second poll interval")
	}
}
