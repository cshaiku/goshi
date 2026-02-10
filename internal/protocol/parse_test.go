package protocol

import (
	"testing"
)

// TestParseFileRequestValid tests parsing a valid file request
func TestParseFileRequestValid(t *testing.T) {
	rawJSON := `{"requested_files":[{"path":"file.txt","reason":"needed"}]}`
	allowed := []string{"file.txt"}

	req, err := ParseFileRequest(rawJSON, allowed)
	if err != nil {
		t.Errorf("expected valid request to parse, got error: %v", err)
	}
	if req == nil {
		t.Errorf("expected non-nil request")
	}
	if len(req.RequestedFiles) != 1 {
		t.Errorf("expected 1 file, got %d", len(req.RequestedFiles))
	}
	if req.RequestedFiles[0].Path != "file.txt" {
		t.Errorf("expected path 'file.txt', got '%s'", req.RequestedFiles[0].Path)
	}
}

// TestParseFileRequestInvalidJSON tests that invalid JSON is rejected
func TestParseFileRequestInvalidJSON(t *testing.T) {
	rawJSON := `{invalid json}`
	allowed := []string{"file.txt"}

	req, err := ParseFileRequest(rawJSON, allowed)
	if err == nil {
		t.Errorf("expected invalid JSON to fail, got request: %v", req)
	}
}

// TestParseFileRequestNotInManifest tests that files not in manifest are rejected
func TestParseFileRequestNotInManifest(t *testing.T) {
	rawJSON := `{"requested_files":[{"path":"file.txt","reason":"needed"}]}`
	allowed := []string{"other.txt"}

	req, err := ParseFileRequest(rawJSON, allowed)
	if err == nil {
		t.Errorf("expected file not in manifest to fail, got request: %v", req)
	}
}

// TestParseFileRequestEmptyRequest tests parsing empty request
func TestParseFileRequestEmptyRequest(t *testing.T) {
	rawJSON := `{"requested_files":[]}`
	allowed := []string{"file.txt"}

	req, err := ParseFileRequest(rawJSON, allowed)
	if err != nil {
		t.Errorf("expected empty request to parse, got error: %v", err)
	}
	if len(req.RequestedFiles) != 0 {
		t.Errorf("expected 0 files in empty request, got %d", len(req.RequestedFiles))
	}
}

// TestParseFileRequestMultipleFiles tests parsing multiple files
func TestParseFileRequestMultipleFiles(t *testing.T) {
	rawJSON := `{"requested_files":[{"path":"file1.txt","reason":"needed"},{"path":"file2.txt","reason":"needed"}]}`
	allowed := []string{"file1.txt", "file2.txt"}

	req, err := ParseFileRequest(rawJSON, allowed)
	if err != nil {
		t.Errorf("expected multiple files to parse, got error: %v", err)
	}
	if len(req.RequestedFiles) != 2 {
		t.Errorf("expected 2 files, got %d", len(req.RequestedFiles))
	}
}

// TestParseFileRequestPartialManifest tests that partial manifest list is handled
func TestParseFileRequestPartialManifest(t *testing.T) {
	rawJSON := `{"requested_files":[{"path":"file1.txt","reason":"needed"}]}`
	// Only file1.txt is allowed
	allowed := []string{"file1.txt"}

	req, err := ParseFileRequest(rawJSON, allowed)
	if err != nil {
		t.Errorf("expected file in manifest to parse, got error: %v", err)
	}
	if len(req.RequestedFiles) != 1 {
		t.Errorf("expected 1 file, got %d", len(req.RequestedFiles))
	}
}

// TestParseFileRequestEmptyManifest tests parsing when manifest is empty
func TestParseFileRequestEmptyManifest(t *testing.T) {
	rawJSON := `{"requested_files":[{"path":"file.txt","reason":"needed"}]}`
	allowed := []string{}

	req, err := ParseFileRequest(rawJSON, allowed)
	if err == nil {
		t.Errorf("expected file not in empty manifest to fail, got request: %v", req)
	}
}

// TestParseFileRequestMissingPath tests handling of missing path field
func TestParseFileRequestMissingReason(t *testing.T) {
	rawJSON := `{"requested_files":[{"path":"file.txt"}]}`
	allowed := []string{"file.txt"}

	req, err := ParseFileRequest(rawJSON, allowed)
	if err != nil {
		t.Errorf("expected request with missing reason to parse, got error: %v", err)
	}
	if len(req.RequestedFiles) != 1 {
		t.Errorf("expected 1 file, got %d", len(req.RequestedFiles))
	}
}
