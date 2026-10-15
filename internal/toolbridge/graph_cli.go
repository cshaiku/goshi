// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const graphUsage = `Shared graph workflows (protocol 1.1.0)
Usage: TOOL tools graph capabilities
       TOOL tools graph schema --kind workflow|plan|run|verification
       TOOL tools graph plan --repo PATH --workflow FILE [--loom REPORT] [--clyde-manifest MANIFEST] [--impact REPORT]
       TOOL tools graph explain (--plan FILE | --run-dir DIRECTORY) [--format text|json|mermaid]
       TOOL tools graph context --plan FILE --stage investigate|implement|verify
       TOOL tools graph run --plan FILE --run-dir NEW_DIRECTORY --approve-digest SHA256 [--allow-ai]
       TOOL tools graph resume --plan FILE --run-dir DIRECTORY --approve-digest SHA256 [--allow-ai]
       TOOL tools graph verify --run-dir DIRECTORY
Planning/context/verification never execute commands. JSON is the default, except explain (text).
Run executes exact approved argv with your privileges; it is not a sandbox. AI requires --allow-ai.
Goshi execution remains limited to its own doctor/version diagnostics. Other native tool approvals still apply.
`

// GraphCommand exposes a common contract to CLI and read-only MCP adapters.
func GraphCommand(args []string, caller, version string, out, errout io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprint(out, graphUsage)
		return 0
	}
	op := args[0]
	if op == "capabilities" {
		if len(args) != 1 {
			return graphCLIError(errout, "invalid_arguments", errors.New("capabilities accepts no flags"), 2)
		}
		_ = graphEmit(out, map[string]any{
			"schema": "starlit.capabilities.v1", "protocol_version": GraphVersion, "bridge_version": Version, "tool": caller, "tool_version": version,
			"operations": []string{"schema", "plan", "explain", "context", "run", "resume", "verify"}, "imports": []string{"loom", "clyde", "vigil"}, "formats": []string{"json", "text", "mermaid"},
			"limits":    map[string]int{"nodes": 128, "attempts_per_fingerprint": 3, "imports": 16, "inputs": 4096, "json_bytes": graphJSONLimit, "file_bytes": graphFileLimit, "total_input_bytes": graphTotalLimit, "stdout_bytes": outputLimit, "stderr_bytes": 64 << 10, "context_bytes": 32768},
			"contracts": map[string]string{"workflow": "starlit.workflow.v1", "plan": "starlit.plan.v1", "run": "starlit.run.v1", "context": "starlit.context.v1", "verification": "starlit.verification.v1", "error": "starlit.error.v1"},
			"execution": "exact argv; no shell interpolation by the runner; commands are not sandboxed", "ai": "explicit allow-ai; advisory only", "goshi": "doctor/version diagnostics only", "exit_codes": map[string]int{"success": 0, "failed_or_stale": 1, "invalid_request": 2, "cancelled": 130},
		})
		return 0
	}
	fs := flag.NewFlagSet("graph "+op, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	repo := fs.String("repo", ".", "repository")
	workflow := fs.String("workflow", "", "workflow JSON")
	planPath := fs.String("plan", "", "saved plan")
	dir := fs.String("run-dir", "", "run directory")
	approval := fs.String("approve-digest", "", "reviewed digest")
	allowAI := fs.Bool("allow-ai", false, "allow explicitly declared AI nodes")
	stage := fs.String("stage", "investigate", "context stage")
	format := fs.String("format", "text", "explanation format")
	kind := fs.String("kind", "workflow", "schema kind")
	loom := fs.String("loom", "", "Loom graph JSON")
	clyde := fs.String("clyde-manifest", "", "Clyde manifest JSON")
	impact := fs.String("impact", "", "Vigil impact JSON")
	if e := fs.Parse(args[1:]); e != nil {
		return graphCLIError(errout, "invalid_arguments", e, 2)
	}
	if fs.NArg() != 0 {
		return graphCLIError(errout, "invalid_arguments", errors.New("unexpected positional arguments"), 2)
	}
	allowed := map[string]string{"schema": "kind", "plan": "repo workflow loom clyde-manifest impact", "explain": "plan run-dir format", "context": "plan stage", "run": "plan run-dir approve-digest allow-ai", "resume": "plan run-dir approve-digest allow-ai", "verify": "run-dir"}
	list, ok := allowed[op]
	if !ok {
		return graphCLIError(errout, "invalid_operation", errors.New(graphUsage), 2)
	}
	var flagErr error
	fs.Visit(func(f *flag.Flag) {
		found := false
		for _, name := range strings.Fields(list) {
			found = found || f.Name == name
		}
		if !found {
			flagErr = fmt.Errorf("flag --%s is not valid for %s", f.Name, op)
		}
	})
	if flagErr != nil {
		return graphCLIError(errout, "invalid_arguments", flagErr, 2)
	}
	if op == "schema" {
		schema, e := graphSchema(*kind)
		if e != nil {
			return graphCLIError(errout, "invalid_schema", e, 2)
		}
		if e = graphEmit(out, schema); e != nil {
			return graphCLIError(errout, "output_error", e, 1)
		}
		return 0
	}
	if op == "plan" {
		if *workflow == "" {
			return graphCLIError(errout, "invalid_arguments", errors.New("--workflow is required"), 2)
		}
		imports := []GraphImport{}
		for _, s := range []GraphImport{{"loom", *loom}, {"clyde", *clyde}, {"vigil", *impact}} {
			if s.Path != "" {
				imports = append(imports, s)
			}
		}
		p, e := BuildGraphPlan(*repo, *workflow, imports)
		if e != nil {
			return graphCLIError(errout, "invalid_plan", e, 2)
		}
		if e = graphEmit(out, p); e != nil {
			return graphCLIError(errout, "output_error", e, 1)
		}
		return 0
	}
	if op == "verify" {
		if *dir == "" {
			return graphCLIError(errout, "invalid_arguments", errors.New("--run-dir is required"), 2)
		}
		v, e := VerifyGraph(*dir)
		if e != nil {
			v.Diagnostic = e.Error()
		}
		if writeErr := graphEmit(out, v); writeErr != nil {
			return graphCLIError(errout, "output_error", writeErr, 1)
		}
		if e != nil {
			return 1
		}
		return 0
	}
	var runView *GraphRun
	if op == "explain" && *dir != "" {
		if *planPath != "" {
			return graphCLIError(errout, "invalid_arguments", errors.New("choose --plan or --run-dir"), 2)
		}
		canonical, err := graphExternalFile(*dir)
		if err != nil {
			return graphCLIError(errout, "invalid_run", err, 2)
		}
		var state GraphRun
		if err = graphDecode(filepath.Join(canonical, "state.json"), &state); err != nil {
			return graphCLIError(errout, "invalid_run", err, 2)
		}
		if len(state.PlanDigest) != 64 || strings.ContainsAny(state.PlanDigest, "/\\") {
			return graphCLIError(errout, "invalid_run", errors.New("invalid run plan digest"), 2)
		}
		runView = &state
		*planPath = filepath.Join(canonical, "plan-"+state.PlanDigest+".json")
	}
	if *planPath == "" {
		return graphCLIError(errout, "invalid_arguments", errors.New("--plan is required"), 2)
	}
	p, e := graphLoadPlan(*planPath)
	if e != nil {
		return graphCLIError(errout, "invalid_plan", e, 2)
	}
	switch op {
	case "explain":
		switch *format {
		case "json":
			if runView != nil {
				e = graphEmit(out, runView)
			} else {
				e = graphEmit(out, p)
			}
		case "text":
			fmt.Fprintf(out, "Workflow %s\nPlan %s\nRepository %s\n%d input files; %d knowledge entities; %d relationships\n", p.Workflow.Name, p.Digest, p.Root, len(p.Files), len(p.Entities), len(p.Links))
			nodes := map[string]GraphStep{}
			for _, n := range p.Workflow.Nodes {
				nodes[n.ID] = n
			}
			for _, id := range p.Order {
				n := nodes[id]
				cmd, _ := json.Marshal(n.Argv)
				if n.Kind == "ai" {
					cmd, _ = json.Marshal([]string{"bram", "ask", "--peer", n.Peer, "--input", n.Report})
				}
				fmt.Fprintf(out, "- %s [%s] needs=%v attempts=%d replay_safe=%t\n  argv: %s\n", id, n.Kind, n.Needs, n.MaxAttempts, n.ReplaySafe, cmd)
			}
			if runView != nil {
				fmt.Fprintf(out, "Run status: %s; qualification: %s\n", runView.Status, runView.Qualification)
				for _, id := range p.Order {
					node := runView.Nodes[id]
					fmt.Fprintf(out, "  %s: %s reused=%t attempts=%d — %s\n", id, node.Status, node.Reused, node.Attempts, node.Reason)
				}
			}
			fmt.Fprintln(out, "Saved snapshot; use verify for current freshness.")
			for _, w := range p.Warnings {
				fmt.Fprintln(out, "Warning: "+w)
			}
		case "mermaid":
			fmt.Fprintln(out, "flowchart TD")
			for i, n := range p.Workflow.Nodes {
				label, _ := json.Marshal(n.ID + " (" + n.Kind + ")")
				fmt.Fprintf(out, "  n%d[%s]\n", i, label)
				for _, dep := range n.Needs {
					for j, other := range p.Workflow.Nodes {
						if other.ID == dep {
							fmt.Fprintf(out, "  n%d --> n%d\n", j, i)
						}
					}
				}
			}
		default:
			e = errors.New("explain format must be text, json or mermaid")
		}
	case "context":
		var packet any
		packet, e = GraphContextPacket(p, *stage)
		if e == nil {
			e = graphEmit(out, packet)
		}
	case "run", "resume":
		if *dir == "" {
			return graphCLIError(errout, "invalid_arguments", errors.New("--run-dir is required"), 2)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		state, runErr := RunGraph(ctx, p, *dir, *approval, caller, op == "resume", *allowAI)
		if state.Schema != "" {
			if writeErr := graphEmit(out, state); writeErr != nil {
				return graphCLIError(errout, "output_error", writeErr, 1)
			}
		}
		if runErr != nil {
			code := 1
			if ctx.Err() != nil {
				code = 130
			}
			return graphCLIError(errout, "execution_failed", runErr, code)
		}
		return 0
	}
	if e != nil {
		return graphCLIError(errout, "operation_failed", e, 1)
	}
	return 0
}
func graphCLIError(out io.Writer, code string, e error, status int) int {
	_ = graphEmit(out, map[string]any{"schema": "starlit.error.v1", "code": code, "message": e.Error()})
	return status
}

func graphSchema(kind string) (any, error) {
	var t reflect.Type
	schemaName := ""
	switch kind {
	case "workflow":
		t = reflect.TypeOf(GraphWorkflow{})
		schemaName = "starlit.workflow.v1"
	case "plan":
		t = reflect.TypeOf(GraphPlan{})
		schemaName = "starlit.plan.v1"
	case "run":
		t = reflect.TypeOf(GraphRun{})
		schemaName = "starlit.run.v1"
	case "verification":
		t = reflect.TypeOf(GraphVerification{})
		schemaName = "starlit.verification.v1"
	default:
		return nil, errors.New("schema kind must be workflow, plan, run or verification")
	}
	schema := graphTypeSchema(t)
	schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	schema["title"] = schemaName
	schema["description"] = "Structural wire schema. The CLI also validates IDs, references, dependency cycles, bounds and operation-specific invariants."
	properties := schema["properties"].(map[string]any)
	properties["schema"] = map[string]any{"const": schemaName}
	return schema, nil
}
func graphTypeSchema(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice:
		return map[string]any{"type": []string{"array", "null"}, "items": graphTypeSchema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": graphTypeSchema(t.Elem())}
	case reflect.Struct:
		props := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			name := tag[0]
			if name == "" {
				name = f.Name
			}
			props[name] = graphTypeSchema(f.Type)
			if !strings.Contains(f.Tag.Get("json"), "omitempty") {
				required = append(required, name)
			}
		}
		sort.Strings(required)
		return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
	default:
		return map[string]any{}
	}
}
