// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGraphToolInputsInvalidateOnlyTheirConsumers(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	toolA := filepath.Join(root, "a.exe")
	toolB := filepath.Join(root, "b.exe")
	runtime := filepath.Join(root, "interpreter.exe")
	for _, p := range []string{toolA, toolB, runtime} {
		graphTestWrite(t, p, []byte("owned fingerprint fixture"))
		os.Chmod(p, 0700)
	}
	graphTestWrite(t, filepath.Join(root, "config.json"), []byte(`{"value":1}`))
	external := filepath.Join(t.TempDir(), "new-dir", "optional.json")
	w := GraphWorkflow{Schema: "starlit.workflow.v1", Name: "tool-freshness", ToolInputs: []GraphToolInputs{{Name: toolA, Files: []string{"config.json"}, OptionalFiles: []string{external}, Runtimes: []string{runtime}, Environment: []string{"STARLIT_TOOL_TEST"}}}, Nodes: []GraphStep{
		{ID: "a", Kind: "check", Argv: []string{toolA}, TimeoutSeconds: 10, MaxAttempts: 1, ReplaySafe: true},
		{ID: "b", Kind: "check", Argv: []string{toolB}, TimeoutSeconds: 10, MaxAttempts: 1, ReplaySafe: true},
		{ID: "downstream", Kind: "check", Argv: []string{toolB}, Needs: []string{"a"}, TimeoutSeconds: 10, MaxAttempts: 1, ReplaySafe: true}}}
	path := filepath.Join(root, "workflow.json")
	graphTestJSON(t, path, w)
	t.Setenv("STARLIT_TOOL_TEST", "secret-one")
	p := graphTestPlan(t, root, path)
	changed := func(label string) {
		t.Helper()
		q := graphTestPlan(t, root, path)
		if q.NodeFingerprints["a"] == p.NodeFingerprints["a"] || q.NodeFingerprints["downstream"] == p.NodeFingerprints["downstream"] || q.NodeFingerprints["b"] != p.NodeFingerprints["b"] {
			t.Fatalf("%s invalidation incorrect", label)
		}
		p = q
	}
	graphTestWrite(t, filepath.Join(root, "config.json"), []byte(`{"value":2}`))
	changed("configuration")
	graphTestWrite(t, runtime, []byte("interpreter changed"))
	changed("runtime binary")
	t.Setenv("STARLIT_TOOL_TEST", "secret-two")
	changed("tool environment")
	os.MkdirAll(filepath.Dir(external), 0700)
	graphTestWrite(t, external, []byte("new optional configuration"))
	changed("optional input appeared")
	os.Remove(external)
	changed("optional input removed")
}
func TestGraphNativeConfigurationFreshness(t *testing.T) {
	root, path, w, binary := graphNativeFixture(t)
	graphTestWrite(t, filepath.Join(root, "sdk-version.txt"), []byte("SDK-1"))
	w.ToolInputs = []GraphToolInputs{{Name: binary, Files: []string{"sdk-version.txt"}, Runtimes: []string{binary}}}
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	graphTestRun(t, p, dir, false)
	graphTestWrite(t, filepath.Join(root, "sdk-version.txt"), []byte("SDK-2"))
	if v, e := VerifyGraph(dir); e == nil || v.Status != "stale" {
		t.Fatal("native SDK identity change was reused", v, e)
	}
	q := graphTestPlan(t, root, path)
	r := graphTestRun(t, q, dir, true)
	if r.Nodes["contract"].Reused {
		t.Fatal("SDK invalidation did not rerun its consumer")
	}
}
