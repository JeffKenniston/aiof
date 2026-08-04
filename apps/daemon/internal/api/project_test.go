package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectHandlers(t *testing.T) {
	server, _ := setupTestServer()

	// mock home dir or tmp dir for aiof config
	tmpDir, err := os.MkdirTemp("", "aiof_project_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override HOME to avoid writing to real ~/.aiof
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	server.activeProjectDir = tmpDir

	// Test /api/projects
	req := httptest.NewRequest("GET", "/api/projects", nil)
	rr := httptest.NewRecorder()
	server.handleProjects(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleProjects expected OK, got %v", rr.Code)
	}

	// Test /api/projects/create
	createReq := map[string]string{
		"name":       "testproj",
		"parentPath": tmpDir,
	}
	body, _ := json.Marshal(createReq)
	req = httptest.NewRequest("POST", "/api/projects/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.handleProjectsCreate(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleProjectsCreate expected OK, got %v", rr.Code)
	}

	// Test /api/projects/load
	loadReq := map[string]string{
		"path": filepath.Join(tmpDir, "testproj"),
	}
	body, _ = json.Marshal(loadReq)
	req = httptest.NewRequest("POST", "/api/projects/load", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.handleProjectsLoad(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleProjectsLoad expected OK, got %v", rr.Code)
	}

	// Test /api/settings/sync
	syncReq := map[string]interface{}{
		"theme": "dark",
	}
	body, _ = json.Marshal(syncReq)
	req = httptest.NewRequest("POST", "/api/settings/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.handleSettingsSync(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleSettingsSync expected OK, got %v", rr.Code)
	}

	// Test /api/settings/load
	req = httptest.NewRequest("GET", "/api/settings/load", nil)
	rr = httptest.NewRecorder()
	server.handleSettingsLoad(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleSettingsLoad expected OK, got %v", rr.Code)
	}
}
