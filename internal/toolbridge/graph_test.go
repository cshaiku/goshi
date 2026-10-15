// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func graphTestWrite(t *testing.T, path string, b []byte) {
	t.Helper()
	if e := os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func graphTestJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	graphTestWrite(t, path, b)
}
func graphTestFixture(t *testing.T) (string, string, GraphWorkflow) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture commands")
	}
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	tool := filepath.Join(root, "fake-tool")
	graphTestWrite(t, tool, []byte("#!/bin/sh\nprintf '%s' '{\"value\":9223372036854775807}'\n"))
	if e = os.Chmod(tool, 0700); e != nil {
		t.Fatal(e)
	}
	graphTestWrite(t, filepath.Join(root, "a.txt"), []byte("a"))
	graphTestWrite(t, filepath.Join(root, "b.txt"), []byte("b"))
	graphTestWrite(t, filepath.Join(root, "owner.md"), []byte("owner guidance"))
	w := GraphWorkflow{Schema: "starlit.workflow.v1", Name: "fixture", Context: []GraphContext{{Path: "owner.md", Stages: []string{"investigate", "verify"}}}, Nodes: []GraphStep{
		{ID: "a", Kind: "check", Argv: []string{tool}, Inputs: []string{"a.txt"}, TimeoutSeconds: 10, MaxAttempts: 3, ReplaySafe: true},
		{ID: "b", Kind: "check", Argv: []string{tool}, Needs: []string{"a"}, TimeoutSeconds: 10, MaxAttempts: 3, ReplaySafe: true},
		{ID: "c", Kind: "analysis", Argv: []string{tool}, Inputs: []string{"b.txt"}, TimeoutSeconds: 10, MaxAttempts: 3, ReplaySafe: true},
	}}
	path := filepath.Join(root, "workflow.json")
	graphTestJSON(t, path, w)
	return root, path, w
}
func graphTestPlan(t *testing.T, root, path string, imports ...GraphImport) GraphPlan {
	t.Helper()
	p, e := BuildGraphPlan(root, path, imports)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func graphTestRun(t *testing.T, p GraphPlan, dir string, resume bool) GraphRun {
	t.Helper()
	r, e := RunGraph(context.Background(), p, dir, p.Digest, "vigil", resume, false)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestGraphConnectedResumeAndSelectiveInvalidation(t *testing.T) {
	root, path, _ := graphTestFixture(t)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	first := graphTestRun(t, p, dir, false)
	if first.Qualification != "declared_checks_passed" {
		t.Fatal(first)
	}
	second := graphTestRun(t, p, dir, true)
	for id, n := range second.Nodes {
		if !n.Reused || n.Attempts != first.Nodes[id].Attempts {
			t.Fatalf("not reused: %+v", n)
		}
	}
	graphTestWrite(t, filepath.Join(root, "a.txt"), []byte("new"))
	changed := graphTestPlan(t, root, path)
	if changed.NodeFingerprints["a"] == p.NodeFingerprints["a"] || changed.NodeFingerprints["b"] == p.NodeFingerprints["b"] || changed.NodeFingerprints["c"] != p.NodeFingerprints["c"] {
		t.Fatal("incorrect invalidation closure")
	}
	third := graphTestRun(t, changed, dir, true)
	if third.Nodes["a"].Reused || third.Nodes["b"].Reused || !third.Nodes["c"].Reused {
		t.Fatal("unaffected work was not preserved")
	}
	v, e := VerifyGraph(dir)
	if e != nil || !v.Consistent || !v.Fresh || v.Qualification != "declared_checks_passed" {
		t.Fatalf("verification: %+v %v", v, e)
	}
	for _, name := range third.Nodes["a"].Artifacts {
		var a GraphAttempt
		graphDecode(filepath.Join(dir, name), &a)
		b, e := os.ReadFile(filepath.Join(dir, a.Stdout))
		if e != nil || string(b) != `{"value":9223372036854775807}` {
			t.Fatal("raw numeric output changed")
		}
	}
}
func TestGraphCommandChangeDoesNotInvalidateIndependentNodes(t *testing.T) {
	root, path, w := graphTestFixture(t)
	p := graphTestPlan(t, root, path)
	w.Nodes[0].Argv = append(w.Nodes[0].Argv, "new-arg")
	graphTestJSON(t, path, w)
	q := graphTestPlan(t, root, path)
	if p.NodeFingerprints["a"] == q.NodeFingerprints["a"] || p.NodeFingerprints["b"] == q.NodeFingerprints["b"] || p.NodeFingerprints["c"] != q.NodeFingerprints["c"] {
		t.Fatal("command invalidation is not selective")
	}
}
func TestGraphStaleApprovalAndTamperedEvidence(t *testing.T) {
	root, path, _ := graphTestFixture(t)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	if _, e := RunGraph(context.Background(), p, dir, "wrong", "vigil", false, false); e == nil {
		t.Fatal("accepted wrong approval")
	}
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("rejected approval wrote files")
	}
	r := graphTestRun(t, p, dir, false)
	var a GraphAttempt
	graphDecode(filepath.Join(dir, r.Nodes["a"].Artifacts[0]), &a)
	graphTestWrite(t, filepath.Join(dir, a.Stdout), []byte("tampered"))
	if _, e := VerifyGraph(dir); e == nil {
		t.Fatal("accepted corrupt output")
	}
	if _, e := RunGraph(context.Background(), p, dir, p.Digest, "vigil", true, false); e == nil {
		t.Fatal("reused corrupt output")
	}
	graphTestWrite(t, filepath.Join(root, "a.txt"), []byte("changed"))
	if _, e := RunGraph(context.Background(), p, filepath.Join(t.TempDir(), "new"), p.Digest, "vigil", false, false); e == nil {
		t.Fatal("accepted stale plan")
	}
}
func TestGraphFailureBudgetsAndDependencyBlocking(t *testing.T) {
	root, path, w := graphTestFixture(t)
	w.Nodes[0].Argv = []string{"/bin/sh", "-c", "printf broken; exit 1"}
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	r, e := RunGraph(context.Background(), p, dir, p.Digest, "loom", false, false)
	if e == nil || r.Nodes["a"].Attempts != 2 || r.Nodes["b"].Status != "blocked" || r.Nodes["c"].Status != "passed" {
		t.Fatalf("failure handling: %+v %v", r, e)
	}
	if !strings.Contains(r.Nodes["a"].Reason, "identical") {
		t.Fatal("missing loop diagnosis")
	}
}
func TestGraphInterruptedNonReplayableRequiresReview(t *testing.T) {
	root, path, w := graphTestFixture(t)
	w.Nodes = w.Nodes[:1]
	w.Nodes[0].ReplaySafe = false
	w.Nodes[0].MaxAttempts = 1
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	graphTestRun(t, p, dir, false)
	var r GraphRun
	graphDecode(filepath.Join(dir, "state.json"), &r)
	ns := r.Nodes["a"]
	ns.Status = "running"
	r.Nodes["a"] = ns
	r.Status = "running"
	graphSave(dir, &r)
	if _, e := RunGraph(context.Background(), p, dir, p.Digest, "vigil", true, false); e == nil || !strings.Contains(e.Error(), "manual recovery") {
		t.Fatal("interrupted effect was replayed", e)
	}
}
func TestGraphInterruptedReplaySafeResumesWithBudget(t *testing.T) {
	root, path, w := graphTestFixture(t)
	w.Nodes = w.Nodes[:1]
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	r := graphTestRun(t, p, dir, false)
	ns := r.Nodes["a"]
	ns.Status = "running"
	r.Nodes["a"] = ns
	r.Status = "running"
	graphSave(dir, &r)
	resumed := graphTestRun(t, p, dir, true)
	if resumed.Nodes["a"].Attempts != 2 {
		t.Fatal("interrupted attempt budget reset")
	}
}
func TestGraphCycleUnknownAndBounds(t *testing.T) {
	for _, kind := range []string{"cycle", "unknown", "unsafe_retry", "escape", "schema", "unknown_field", "oversized", "special_file"} {
		t.Run(kind, func(t *testing.T) {
			root, path, w := graphTestFixture(t)
			switch kind {
			case "cycle":
				w.Nodes[0].Needs = []string{"b"}
			case "unknown":
				w.Nodes[0].Needs = []string{"missing"}
			case "unsafe_retry":
				w.Nodes[0].ReplaySafe = false
			case "escape":
				w.Inputs = []string{"../secret"}
			case "schema":
				w.Schema = "future"
			case "oversized":
				graphTestWrite(t, filepath.Join(root, "a.txt"), bytes.Repeat([]byte("x"), graphFileLimit+1))
			case "special_file":
				w.Inputs = []string{"."}
			}
			graphTestJSON(t, path, w)
			if kind == "unknown_field" {
				b, _ := os.ReadFile(path)
				b = bytes.Replace(b, []byte(`"name":"fixture"`), []byte(`"name":"fixture","typo":true`), 1)
				graphTestWrite(t, path, b)
			}
			if _, e := BuildGraphPlan(root, path, nil); e == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}
func TestGraphContextStagesAndEnvironmentFreshness(t *testing.T) {
	root, path, w := graphTestFixture(t)
	w.Environment = []string{"STARLIT_GRAPH_TEST"}
	graphTestJSON(t, path, w)
	t.Setenv("STARLIT_GRAPH_TEST", "secret-one")
	p := graphTestPlan(t, root, path)
	packet, e := GraphContextPacket(p, "investigate")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(packet)
	if !bytes.Contains(b, []byte(`"text":"owner guidance"`)) || bytes.Contains(b, []byte("secret-one")) {
		t.Fatal("context or privacy failure")
	}
	packet, e = GraphContextPacket(p, "implement")
	if e != nil {
		t.Fatal(e)
	}
	b, _ = json.Marshal(packet)
	if bytes.Contains(b, []byte(`"text":"owner guidance"`)) {
		t.Fatal("stage was ignored")
	}
	t.Setenv("STARLIT_GRAPH_TEST", "secret-two")
	if _, e = GraphContextPacket(p, "investigate"); e == nil {
		t.Fatal("environment change did not invalidate context")
	}
}
func TestGraphLockAndPathIsolation(t *testing.T) {
	root, path, _ := graphTestFixture(t)
	p := graphTestPlan(t, root, path)
	if _, e := RunGraph(context.Background(), p, filepath.Join(root, "run"), p.Digest, "vigil", false, false); e == nil {
		t.Fatal("wrote inside input repository")
	}
	dir := t.TempDir()
	unlock, e := graphLock(dir)
	if e != nil {
		t.Fatal(e)
	}
	if release, e := graphLock(dir); e == nil {
		release()
		t.Fatal("double writer admitted")
	}
	unlock()
	release, e := graphLock(dir)
	if e != nil {
		t.Fatal(e)
	}
	release()
}
func TestGraphTimeoutOutputAndCancellation(t *testing.T) {
	root, path, w := graphTestFixture(t)
	w.Nodes = w.Nodes[:1]
	graphTestWrite(t, filepath.Join(root, "burst.txt"), bytes.Repeat([]byte("x"), outputLimit+1))
	w.Nodes[0].Argv = []string{"/bin/cat", "burst.txt"}
	w.Nodes[0].Inputs = append(w.Nodes[0].Inputs, "burst.txt")
	w.Nodes[0].TimeoutSeconds = 10
	w.Nodes[0].MaxAttempts = 1
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	r := graphExecute(context.Background(), p, w.Nodes[0], 1, "")
	if r.Attempt.Status != "output-limit" || len(r.Out) > outputLimit {
		t.Fatal("output bound failed", r.Attempt)
	}
	w.Nodes[0].Argv = []string{"/bin/sh", "-c", "sleep 5"}
	graphTestJSON(t, path, w)
	p = graphTestPlan(t, root, path)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	r = graphExecute(ctx, p, w.Nodes[0], 1, "")
	if r.Attempt.Status != "cancelled" || time.Since(start) > 3*time.Second {
		t.Fatal("cancellation failed")
	}
}
func TestGraphAIAndGoshiAuthority(t *testing.T) {
	root, path, w := graphTestFixture(t)
	p := graphTestPlan(t, root, path)
	if e := graphPolicy("goshi", p, false); e == nil {
		t.Fatal("Goshi gained general execution")
	}
	w.Nodes = []GraphStep{{ID: "diagnosis", Kind: "ai", Peer: "mock", Prompt: "Explain the failure", Report: "owner.md", TimeoutSeconds: 10, MaxAttempts: 1}}
	graphTestJSON(t, path, w)
	p = graphTestPlan(t, root, path)
	if e := graphPolicy("bram", p, false); e == nil {
		t.Fatal("AI was implicitly authorized")
	}
	if e := graphPolicy("bram", p, true); e != nil {
		t.Fatal(e)
	}
	if e := graphPolicy("goshi", p, true); e == nil {
		t.Fatal("Goshi gained AI execution authority")
	}
}
func TestGraphCLIContractsArePortableAndStrict(t *testing.T) {
	for _, caller := range []string{"vigil", "loom", "clyde", "bram", "nora", "goshi"} {
		var out, errout bytes.Buffer
		handled, code := Handle([]string{"tools", "graph", "capabilities"}, caller, "test", &out, &errout)
		if !handled || code != 0 || !json.Valid(out.Bytes()) {
			t.Fatal(caller, code, errout.String())
		}
	}
	for _, kind := range []string{"workflow", "plan", "run", "verification"} {
		schema, e := graphSchema(kind)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.Marshal(schema)
		if !bytes.Contains(b, []byte("additionalProperties")) {
			t.Fatal("missing strict schema")
		}
	}
	var out, errout bytes.Buffer
	if code := GraphCommand([]string{"verify", "--repo", "."}, "loom", "test", &out, &errout); code != 2 || !json.Valid(errout.Bytes()) {
		t.Fatal("unknown flags not rejected")
	}
}
func TestGraphImportsRejectStaleClydeAndConnectVigil(t *testing.T) {
	root, path, w := graphTestFixture(t)
	fileBytes, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	manifest := filepath.Join(t.TempDir(), "manifest.json")
	graphTestJSON(t, manifest, map[string]any{"schema": "clyde.bundle.v1", "repo": root, "files": []any{map[string]any{"path": "a.txt", "sha256": graphHash(fileBytes)}}})
	p := graphTestPlan(t, root, path, GraphImport{"clyde", manifest})
	if len(p.Entities) != 1 {
		t.Fatal("Clyde source not connected")
	}
	graphTestWrite(t, filepath.Join(root, "a.txt"), []byte("new"))
	if _, e := BuildGraphPlan(root, path, []GraphImport{{"clyde", manifest}}); e == nil {
		t.Fatal("stale Clyde source admitted")
	}
	w.Nodes = nil
	graphTestJSON(t, path, w)
	impact := filepath.Join(t.TempDir(), "impact.json")
	graphTestJSON(t, impact, map[string]any{"schema_version": 1, "nodes": []any{map[string]any{"id": "check:build", "kind": "check", "argv": []string{p.Tools[0].Path}}, map[string]any{"id": "check:render", "kind": "check", "argv": []string{p.Tools[0].Path}}}, "edges": []any{map[string]any{"from": "check:render", "to": "check:build", "relation": "requires"}}, "unmatched": []string{"unrelated"}})
	p = graphTestPlan(t, root, path, GraphImport{"vigil", impact})
	if len(p.Workflow.Nodes) != 2 || p.Order[0] != "impact-build" || len(p.Workflow.Nodes[1].Needs) != 1 {
		t.Fatal("Vigil dependencies lost")
	}
}

func TestGraphFailureDiagnosisIsExplicitAndCannotQualifyFailure(t *testing.T) {
	root, path, w := graphTestFixture(t)
	mock := filepath.Join(root, "bram")
	graphTestWrite(t, mock, []byte("#!/bin/sh\nprintf '%s' '{\"output\":\"diagnosis only\"}'\n"))
	os.Chmod(mock, 0700)
	t.Setenv("PATH", root)
	w.Nodes = w.Nodes[:1]
	w.Nodes[0].Argv = []string{"/bin/sh", "-c", "printf failure; exit 1"}
	w.Nodes[0].MaxAttempts = 1
	w.Nodes = append(w.Nodes, GraphStep{ID: "diagnose", Kind: "ai", Peer: "mock", Prompt: "Explain", ReportFrom: "a", When: "failure", Needs: []string{"a"}, TimeoutSeconds: 10, MaxAttempts: 1})
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	if _, e := RunGraph(context.Background(), p, dir, p.Digest, "bram", false, false); e == nil {
		t.Fatal("implicit AI execution")
	}
	r, e := RunGraph(context.Background(), p, dir, p.Digest, "bram", false, true)
	if e == nil || r.Nodes["a"].Status != "failed" || r.Nodes["diagnose"].Status != "passed" || r.Qualification != "unverified" {
		t.Fatalf("diagnosis falsely qualified failure: %+v %v", r, e)
	}
	// A successful source check skips failure-only diagnosis entirely.
	w.Nodes[0].Argv = []string{"/bin/sh", "-c", "printf success"}
	graphTestJSON(t, path, w)
	p = graphTestPlan(t, root, path)
	r, e = RunGraph(context.Background(), p, filepath.Join(t.TempDir(), "run"), p.Digest, "bram", false, true)
	if e != nil || r.Nodes["diagnose"].Status != "skipped" || r.Nodes["diagnose"].Attempts != 0 {
		t.Fatal("unnecessary diagnosis executed", r, e)
	}
}
func TestGraphLoomSourceHashFreshness(t *testing.T) {
	root, path, _ := graphTestFixture(t)
	source := filepath.Join(root, "a.txt")
	b, _ := os.ReadFile(source)
	report := filepath.Join(t.TempDir(), "loom.json")
	graphTestJSON(t, report, map[string]any{"schema_version": "1", "components": []any{map[string]any{"name": "A", "path": "a.txt"}}, "edges": []any{}, "source_files": []any{map[string]any{"path": "a.txt", "sha256": graphHash(b)}}})
	p := graphTestPlan(t, root, path, GraphImport{"loom", report})
	if len(p.Entities) != 1 {
		t.Fatal("missing imported component")
	}
	graphTestWrite(t, source, []byte("changed"))
	if _, e := BuildGraphPlan(root, path, []GraphImport{{"loom", report}}); e == nil {
		t.Fatal("accepted stale extraction")
	}
}

func TestGraphFinalInputChangeCannotClaimCompletion(t *testing.T) {
	root, path, w := graphTestFixture(t)
	w.Nodes = w.Nodes[:1]
	w.Nodes[0].Argv = []string{"/bin/sh", "-c", "printf changed > a.txt"}
	w.Nodes[0].MaxAttempts = 1
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	r, e := RunGraph(context.Background(), p, filepath.Join(t.TempDir(), "run"), p.Digest, "vigil", false, false)
	if e == nil || r.Status != "stale" || r.Qualification != "unverified" {
		t.Fatal("mutated input falsely qualified", r, e)
	}
}

func TestGraphScopeMembershipInvalidatesAdditions(t *testing.T) {
	root, path, w := graphTestFixture(t)
	os.Mkdir(filepath.Join(root, "ui"), 0700)
	graphTestWrite(t, filepath.Join(root, "ui", "A.qml"), []byte("Item {}"))
	w.Nodes[0].Scopes = []GraphScope{{Root: "ui", Suffixes: []string{".qml"}}}
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	graphTestWrite(t, filepath.Join(root, "ui", "New.qml"), []byte("Rectangle {}"))
	q := graphTestPlan(t, root, path)
	if p.NodeFingerprints["a"] == q.NodeFingerprints["a"] || p.NodeFingerprints["b"] == q.NodeFingerprints["b"] || p.NodeFingerprints["c"] != q.NodeFingerprints["c"] {
		t.Fatal("directory membership not tracked selectively")
	}
}

func TestGraphDiagnosisDoesNotReuseSupersededFailure(t *testing.T) {
	root, path, w := graphTestFixture(t)
	mock := filepath.Join(root, "bram")
	graphTestWrite(t, mock, []byte("#!/bin/sh\nprintf diagnosis\n"))
	os.Chmod(mock, 0700)
	t.Setenv("PATH", root)
	w.Nodes = w.Nodes[:1]
	w.Nodes[0].Argv = []string{"/bin/sh", "-c", "printf failed; exit 1"}
	w.Nodes = append(w.Nodes, GraphStep{ID: "diagnose", Kind: "ai", Peer: "mock", Prompt: "Explain", ReportFrom: "a", When: "failure", Needs: []string{"a"}, TimeoutSeconds: 10, MaxAttempts: 1})
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	first, e := RunGraph(context.Background(), p, dir, p.Digest, "bram", false, true)
	if e == nil || first.Nodes["a"].Attempts != 2 || first.Nodes["diagnose"].Status != "passed" {
		t.Fatal("fixture failure", first, e)
	}
	resumed, e := RunGraph(context.Background(), p, dir, p.Digest, "bram", true, true)
	if e == nil || resumed.Nodes["a"].Attempts != 3 || resumed.Nodes["diagnose"].Status != "blocked" || resumed.Nodes["diagnose"].Reused {
		t.Fatal("stale diagnosis was reused", resumed, e)
	}
}

func TestGraphAIReportBudgetRejectedBeforeExecution(t *testing.T) {
	root, path, w := graphTestFixture(t)
	graphTestWrite(t, filepath.Join(root, "report.txt"), bytes.Repeat([]byte("x"), (1<<20)+1))
	w.Nodes = []GraphStep{{ID: "ai", Kind: "ai", Peer: "mock", Prompt: "Explain", Report: "report.txt", TimeoutSeconds: 10, MaxAttempts: 1}}
	graphTestJSON(t, path, w)
	if _, e := BuildGraphPlan(root, path, nil); e == nil {
		t.Fatal("oversized peer report accepted")
	}
}
