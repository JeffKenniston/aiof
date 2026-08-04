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

func TestFSHandlers(t *testing.T) {
	server, _ := setupTestServer()

	tmpDir, err := os.MkdirTemp("", "aiof_fs_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	server.activeProjectDir = tmpDir

	// Test /api/save
	saveReqBody := map[string]string{
		"filepath": "test.txt",
		"content":  "hello world",
	}
	body, _ := json.Marshal(saveReqBody)
	req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleSave(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleSave expected OK, got %v", rr.Code)
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil || string(content) != "hello world" {
		t.Errorf("failed to verify saved file content")
	}

	// Test /api/read
	req = httptest.NewRequest("GET", "/api/read?filepath=test.txt", nil)
	rr = httptest.NewRecorder()
	server.handleRead(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleRead expected OK, got %v", rr.Code)
	}
	if rr.Body.String() != "hello world" {
		t.Errorf("expected 'hello world', got %v", rr.Body.String())
	}

	// Test /api/rename
	renameReqBody := map[string]string{
		"oldPath": "test.txt",
		"newPath": "test2.txt",
	}
	body, _ = json.Marshal(renameReqBody)
	req = httptest.NewRequest("POST", "/api/rename", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.handleRename(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleRename expected OK, got %v", rr.Code)
	}

	// Test /api/files (buildTree)
	req = httptest.NewRequest("GET", "/api/files", nil)
	rr = httptest.NewRecorder()
	server.handleFiles(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleFiles expected OK, got %v", rr.Code)
	}

	// Test /api/delete
	deleteReqBody := map[string]string{
		"path": "test2.txt",
	}
	body, _ = json.Marshal(deleteReqBody)
	req = httptest.NewRequest("POST", "/api/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.handleDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleDelete expected OK, got %v", rr.Code)
	}

	// Test /api/create_node
	createReqBody := map[string]interface{}{
		"path": "new_dir",
		"isDir": true,
	}
	body, _ = json.Marshal(createReqBody)
	req = httptest.NewRequest("POST", "/api/create_node", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.handleCreateNode(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleCreateNode expected OK, got %v", rr.Code)
	}
}

func TestSanitizePathTraversal(t *testing.T) {
	server, _ := setupTestServer()

	_, err := server.sanitizePath("/base", "../etc/passwd")
	if err == nil {
		t.Errorf("expected error on path traversal")
	}
}
