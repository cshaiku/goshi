package toolbridge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSnapshotReceipt(t *testing.T) {
	data, err := os.ReadFile("UPSTREAM.json")
	if os.IsNotExist(err) {
		return
	} // The authoritative package is not a snapshot.
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Version string            `json:"version"`
		Files   map[string]string `json:"files"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Version != Version || len(receipt.Files) != 3 {
		t.Fatal("invalid snapshot receipt")
	}
	for name, want := range receipt.Files {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != want {
			t.Fatalf("snapshot drift: %s", name)
		}
	}
}

func fakeTool(t *testing.T, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mock executable")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return path
}
func TestStandaloneAndMissingCompanions(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var out, stderr bytes.Buffer
	if handled, _ := Handle([]string{"analyze"}, "nora", "1", &out, &stderr); handled {
		t.Fatal("intercepted native command")
	}
	if handled, code := Handle([]string{"tools", "doctor"}, "nora", "1", &out, &stderr); !handled || code != 0 {
		t.Fatal(stderr.String())
	}
	var report struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Tools) != len(names) {
		t.Fatal("missing inventory entries")
	}
	for _, tool := range report.Tools {
		if tool.Status != "missing" {
			t.Fatalf("unexpected status: %+v", tool)
		}
	}
}
func TestLegacyReceipts(t *testing.T) {
	r := receipt([]byte("source=/old/checkout\nsha256=abc\ncommit: def\nworking_tree_changes:\n M a.go\n"))
	if r["source"] != "/old/checkout" || r["sha256"] != "abc" || r["commit"] != "def" || r["dirty"] != "true" {
		t.Fatal(r)
	}
}
func TestPlanDoesNotExecuteAndValidatesInputs(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	fakeTool(t, "nora", "touch '"+marker+"'")
	input := filepath.Join(t.TempDir(), "access.log")
	os.WriteFile(input, []byte("example"), 0600)
	p, err := buildPlan("log-review", t.TempDir(), "", input, false)
	if err != nil || len(p.Steps) != 1 || p.Inputs[0].SHA256 == "" {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("planning executed a tool")
	}
	if _, err := buildPlan("log-review", ".", "", input, true); err == nil {
		t.Fatal("accepted irrelevant flag")
	}
	if _, err := buildPlan("ui-review", ".", t.TempDir(), "", false); err == nil {
		t.Fatal("accepted directory input")
	}
}
func TestRunPreservesExactNumbersAndRefusesOverwrite(t *testing.T) {
	fakeTool(t, "nora", "printf '%s' '{\"count\":9223372036854775807}'")
	parent := t.TempDir()
	dest := filepath.Join(parent, "run")
	p := Plan{SchemaVersion: 1, Recipe: "log-review", Directory: parent, Inputs: []Input{}, Steps: []Step{{Tool: "nora", Args: []string{"analyze", "example"}}}}
	var out bytes.Buffer
	if err := runPlan(p, dest, "nora", "1", &out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "01-nora.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("9223372036854775807")) {
		t.Fatal("numeric payload was changed")
	}
	var a Artifact
	if err := json.Unmarshal(data, &a); err != nil {
		t.Fatal(err)
	}
	rawHash, e := digest(filepath.Join(dest, a.RawOutputFile))
	if e != nil || rawHash != a.OutputSHA256 {
		t.Fatal("raw output digest mismatch")
	}
	if a.Status != "ok" || a.ExecutableSHA256 == "" || a.OutputSHA256 == "" {
		t.Fatal(a)
	}
	if err := runPlan(p, dest, "nora", "1", &out); err == nil {
		t.Fatal("overwrote existing run")
	}
	link := filepath.Join(parent, "link")
	if err := os.Symlink(dest, link); err != nil {
		t.Fatal(err)
	}
	if err := runPlan(p, link, "nora", "1", &out); err == nil {
		t.Fatal("followed existing destination symlink")
	}
}
func TestFailureAndMissingArePreserved(t *testing.T) {
	fakeTool(t, "nora", "printf 'partial'; printf 'problem' >&2; exit 3")
	a := execute(Step{Tool: "nora"}, t.TempDir(), time.Second)
	if a.Status != "failed" || a.ExitCode != 3 || a.Text != "partial" || a.Stderr != "problem" {
		t.Fatal(a)
	}
	a = execute(Step{Tool: "clyde"}, t.TempDir(), time.Second)
	if a.Status != "missing" {
		t.Fatal(a)
	}
}
func TestTimeoutAndOutputBound(t *testing.T) {
	fakeTool(t, "nora", "while :; do :; done")
	a := execute(Step{Tool: "nora"}, t.TempDir(), 50*time.Millisecond)
	if a.Status != "timeout" {
		t.Fatal(a)
	}
	fakeTool(t, "nora", "while :; do printf '"+strings.Repeat("x", 4096)+"'; done")
	a = execute(Step{Tool: "nora"}, t.TempDir(), 5*time.Second)
	if a.Status != "output-limit" || len(a.Text) > outputLimit {
		t.Fatalf("status %s bytes %d", a.Status, len(a.Text))
	}
}
