// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGraphNativeHelper is an owned subprocess fixture, not a shell or real peer.
func TestGraphNativeHelper(t *testing.T) {
	args := []string{}
	for i, a := range os.Args {
		if a == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "echo":
		fmt.Print(`{"value":9223372036854775807}`)
		os.Exit(0)
	case "flood":
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("x"), outputLimit+1))
		os.Exit(0)
	case "wait":
		if len(args) > 1 {
			_ = os.WriteFile(args[1], []byte("ready"), 0600)
		}
		time.Sleep(2 * time.Second)
		os.Exit(0)
	case "lock":
		release, e := graphLock(args[1])
		if e != nil {
			os.Exit(3)
		}
		defer release()
		_ = os.WriteFile(args[2], []byte("ready"), 0600)
		time.Sleep(20 * time.Second)
	case "orchestrate":
		p, e := graphLoadPlan(args[1])
		if e != nil {
			fmt.Fprint(os.Stderr, e)
			os.Exit(4)
		}
		_, e = RunGraph(context.Background(), p, args[2], p.Digest, "bram", false, false)
		if e != nil {
			fmt.Fprint(os.Stderr, e)
			os.Exit(5)
		}
		os.Exit(0)
	default:
		os.Exit(6)
	}
}
func graphNativeFixture(t *testing.T) (string, string, GraphWorkflow, string) {
	t.Helper()
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	binary, e = filepath.EvalSymlinks(binary)
	if e != nil {
		t.Fatal(e)
	}
	graphTestWrite(t, filepath.Join(root, "data.txt"), []byte("one"))
	step := GraphStep{ID: "contract", Kind: "check", Argv: []string{binary, "-test.run=^TestGraphNativeHelper$", "--", "echo"}, Inputs: []string{"data.txt"}, TimeoutSeconds: 15, MaxAttempts: 3, ReplaySafe: true}
	w := GraphWorkflow{Schema: "starlit.workflow.v1", Name: "native-recovery", BudgetSeconds: 60, Nodes: []GraphStep{step}}
	path := filepath.Join(root, "workflow.json")
	graphTestJSON(t, path, w)
	return root, path, w, binary
}
func graphNativeReady(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if _, e := os.Stat(path); e == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("native helper did not become ready")
}
func TestGraphNativeRunResumeFreshnessAndTamper(t *testing.T) {
	root, path, _, _ := graphNativeFixture(t)
	p := graphTestPlan(t, root, path)
	dir := filepath.Join(t.TempDir(), "run")
	graphTestRun(t, p, dir, false)
	resumed := graphTestRun(t, p, dir, true)
	if !resumed.Nodes["contract"].Reused {
		t.Fatal("native evidence not reused")
	}
	if v, e := VerifyGraph(dir); e != nil || !v.Fresh {
		t.Fatalf("native verification: %+v %v", v, e)
	}
	graphTestWrite(t, filepath.Join(root, "data.txt"), []byte("two"))
	if v, e := VerifyGraph(dir); e == nil || v.Status != "stale" {
		t.Fatal("native stale input accepted")
	}
	q := graphTestPlan(t, root, path)
	resumed = graphTestRun(t, q, dir, true)
	if resumed.Nodes["contract"].Reused {
		t.Fatal("changed input reused")
	}
	var a GraphAttempt
	if e := graphDecode(filepath.Join(dir, resumed.Nodes["contract"].Artifacts[0]), &a); e != nil {
		t.Fatal(e)
	}
	graphTestWrite(t, filepath.Join(dir, a.Stdout), []byte("tampered"))
	if _, e := VerifyGraph(dir); e == nil {
		t.Fatal("native tampering accepted")
	}
}
func TestGraphNativeKernelLockReleasedAfterProcessDeath(t *testing.T) {
	_, _, _, binary := graphNativeFixture(t)
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	cmd := exec.Command(binary, "-test.run=^TestGraphNativeHelper$", "--", "lock", dir, ready)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()
	graphNativeReady(t, ready)
	if release, e := graphLock(dir); e == nil {
		release()
		t.Fatal("second native writer admitted")
	}
	if e := cmd.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	_ = cmd.Wait()
	release, e := graphLock(dir)
	if e != nil {
		t.Fatal("kernel lock survived process death", e, output.String())
	}
	release()
}
func TestGraphNativeInterruptedCheckpointRecovery(t *testing.T) {
	root, path, w, binary := graphNativeFixture(t)
	ready := filepath.Join(t.TempDir(), "child-ready")
	w.Nodes[0].Argv = []string{binary, "-test.run=^TestGraphNativeHelper$", "--", "wait", ready}
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	parent := t.TempDir()
	planPath := filepath.Join(parent, "plan.json")
	graphTestJSON(t, planPath, p)
	dir := filepath.Join(parent, "run")
	cmd := exec.Command(binary, "-test.run=^TestGraphNativeHelper$", "--", "orchestrate", planPath, dir)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()
	graphNativeReady(t, ready)
	if e := cmd.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	_ = cmd.Wait()
	var state GraphRun
	if e := graphDecode(filepath.Join(dir, "state.json"), &state); e != nil {
		t.Fatal(e)
	}
	if state.Nodes["contract"].Status != "running" {
		t.Fatal("running checkpoint not persisted")
	}
	// The owned child exits after two seconds; no long-lived descendant is left behind.
	time.Sleep(2200 * time.Millisecond)
	r := graphTestRun(t, p, dir, true)
	if r.Nodes["contract"].Attempts != 2 {
		t.Fatalf("interrupted attempt was lost: %+v", r)
	}
	if v, e := VerifyGraph(dir); e != nil || !v.Consistent || !v.Fresh {
		t.Fatal("recovered evidence invalid", v, e, output.String())
	}
}
func TestGraphNativeCancellationAndOutputBound(t *testing.T) {
	root, path, w, binary := graphNativeFixture(t)
	w.Nodes[0].Argv = []string{binary, "-test.run=^TestGraphNativeHelper$", "--", "flood"}
	graphTestJSON(t, path, w)
	p := graphTestPlan(t, root, path)
	r := graphExecute(context.Background(), p, w.Nodes[0], 1, "")
	if r.Attempt.Status != "output-limit" || len(r.Out) > outputLimit {
		t.Fatalf("native output bound: %+v", r.Attempt)
	}
	ready := filepath.Join(t.TempDir(), "cancel-ready")
	w.Nodes[0].Argv = []string{binary, "-test.run=^TestGraphNativeHelper$", "--", "wait", ready}
	graphTestJSON(t, path, w)
	p = graphTestPlan(t, root, path)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan graphExecution, 1)
	go func() { done <- graphExecute(ctx, p, w.Nodes[0], 1, "") }()
	graphNativeReady(t, ready)
	cancel()
	select {
	case r = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("active native cancellation did not finish")
	}
	if r.Attempt.Status != "cancelled" || !strings.Contains(r.Attempt.Diagnostic, "canceled") {
		t.Fatalf("native cancellation: %+v", r.Attempt)
	}
}
func TestGraphNativeContractSchemas(t *testing.T) {
	for _, kind := range []string{"workflow", "plan", "run", "verification"} {
		schema, e := graphSchema(kind)
		if e != nil {
			t.Fatal(e)
		}
		b, e := json.Marshal(schema)
		if e != nil || !json.Valid(b) {
			t.Fatal("invalid native schema", kind, e)
		}
	}
}
