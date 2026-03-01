package adt

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGctsListRepositories(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository": newTestResponse(`{"result":[{"rid":"repo1","name":"test-repo","url":"https://git.example.com/repo.git","branch":"main","status":"READY","role":"SOURCE"}]}`),
			"discovery":                      newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	repos, err := client.GctsListRepositories(context.Background())
	if err != nil {
		t.Fatalf("GctsListRepositories failed: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(repos))
	}

	if repos[0].Rid != "repo1" {
		t.Errorf("Rid = %v, want repo1", repos[0].Rid)
	}
	if repos[0].Name != "test-repo" {
		t.Errorf("Name = %v, want test-repo", repos[0].Name)
	}
	if repos[0].Status != "READY" {
		t.Errorf("Status = %v, want READY", repos[0].Status)
	}
}

func TestGctsGetRepository(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository/repo1": newTestResponse(`{"result":{"rid":"repo1","name":"test-repo","url":"https://git.example.com/repo.git","branch":"main","status":"READY","role":"SOURCE","config":[{"key":"VCS_TARGET_DIR","value":"src/"}]}}`),
			"discovery": newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	repo, err := client.GctsGetRepository(context.Background(), "repo1")
	if err != nil {
		t.Fatalf("GctsGetRepository failed: %v", err)
	}

	if repo.Rid != "repo1" {
		t.Errorf("Rid = %v, want repo1", repo.Rid)
	}
	if len(repo.Config) != 1 {
		t.Fatalf("Expected 1 config entry, got %d", len(repo.Config))
	}
	if repo.Config[0].Key != "VCS_TARGET_DIR" {
		t.Errorf("Config key = %v, want VCS_TARGET_DIR", repo.Config[0].Key)
	}
}

func TestGctsCreateRepository(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository": newTestResponse(`{"result":{"rid":"new-repo","name":"new-repo","url":"https://git.example.com/new.git","branch":"main","status":"CREATED","role":"SOURCE"}}`),
			"discovery":                      newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	repo, err := client.GctsCreateRepository(context.Background(), GctsCreateOptions{
		Rid:  "new-repo",
		Name: "new-repo",
		URL:  "https://git.example.com/new.git",
	})
	if err != nil {
		t.Fatalf("GctsCreateRepository failed: %v", err)
	}

	if repo.Rid != "new-repo" {
		t.Errorf("Rid = %v, want new-repo", repo.Rid)
	}
	if repo.Status != "CREATED" {
		t.Errorf("Status = %v, want CREATED", repo.Status)
	}
}

func TestGctsDeleteRepository(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository/repo1": {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     http.Header{"X-CSRF-Token": []string{"test-token"}},
			},
			"discovery": newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	err := client.GctsDeleteRepository(context.Background(), "repo1")
	if err != nil {
		t.Fatalf("GctsDeleteRepository failed: %v", err)
	}
}

func TestGctsCloneRepository(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository/repo1/clone": newTestResponse(`{"result":{}}`),
			"discovery": newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	err := client.GctsCloneRepository(context.Background(), "repo1")
	if err != nil {
		t.Fatalf("GctsCloneRepository failed: %v", err)
	}
}

func TestGctsPull(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository/repo1/pullByCommit": newTestResponse(`{"result":{"fromCommit":"abc123","toCommit":"def456"}}`),
			"discovery": newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	result, err := client.GctsPull(context.Background(), "repo1", "def456")
	if err != nil {
		t.Fatalf("GctsPull failed: %v", err)
	}

	if result.FromCommit != "abc123" {
		t.Errorf("FromCommit = %v, want abc123", result.FromCommit)
	}
	if result.ToCommit != "def456" {
		t.Errorf("ToCommit = %v, want def456", result.ToCommit)
	}
}

func TestGctsCommit(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository/repo1/commit": newTestResponse(`{"result":{"id":"abc123","message":"test commit"}}`),
			"discovery": newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	result, err := client.GctsCommit(context.Background(), "repo1", GctsCommitOptions{
		Message: "test commit",
	})
	if err != nil {
		t.Fatalf("GctsCommit failed: %v", err)
	}

	if result.CommitID != "abc123" {
		t.Errorf("CommitID = %v, want abc123", result.CommitID)
	}
}

func TestGctsListBranches(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository/repo1/branches": newTestResponse(`{"result":[{"name":"main","type":"branch","isActive":true},{"name":"develop","type":"branch","isActive":false}]}`),
			"discovery": newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	branches, err := client.GctsListBranches(context.Background(), "repo1")
	if err != nil {
		t.Fatalf("GctsListBranches failed: %v", err)
	}

	if len(branches) != 2 {
		t.Fatalf("Expected 2 branches, got %d", len(branches))
	}

	if branches[0].Name != "main" {
		t.Errorf("Branch name = %v, want main", branches[0].Name)
	}
	if !branches[0].IsActive {
		t.Error("Expected main branch to be active")
	}
}

func TestGctsSwitchBranch(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository/repo1/switchBranch": newTestResponse(`{}`),
			"discovery": newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	err := client.GctsSwitchBranch(context.Background(), "repo1", "develop")
	if err != nil {
		t.Fatalf("GctsSwitchBranch failed: %v", err)
	}
}

func TestGctsGetHistory(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository/repo1/getHistory": newTestResponse(`{"result":[{"id":"abc123","message":"initial commit","author":"user","date":"2025-01-01"},{"id":"def456","message":"second commit","author":"user","date":"2025-01-02"}]}`),
			"discovery": newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	history, err := client.GctsGetHistory(context.Background(), "repo1")
	if err != nil {
		t.Fatalf("GctsGetHistory failed: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("Expected 2 commits, got %d", len(history))
	}

	if history[0].ID != "abc123" {
		t.Errorf("Commit ID = %v, want abc123", history[0].ID)
	}
}

func TestGctsErrorLogParsing(t *testing.T) {
	mock := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/cts_abapvcs/repository": newTestResponse(`{"result":null,"errorLog":[{"severity":"error","message":"Repository not found"},{"severity":"error","message":"Check configuration"}]}`),
			"discovery":                      newTestResponse("OK"),
		},
	}

	cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
	transport := NewTransportWithClient(cfg, mock)
	client := NewClientWithTransport(cfg, transport)

	_, err := client.GctsListRepositories(context.Background())
	if err == nil {
		t.Fatal("Expected error from errorLog, got nil")
	}

	if !strings.Contains(err.Error(), "Repository not found") {
		t.Errorf("Error should contain 'Repository not found', got: %v", err)
	}
	if !strings.Contains(err.Error(), "Check configuration") {
		t.Errorf("Error should contain 'Check configuration', got: %v", err)
	}
}
