// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"
)

const GraphVersion = "1.1.0"
const graphJSONLimit = 8 << 20
const graphFileLimit = 8 << 20
const graphTotalLimit = 64 << 20

var graphID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,95}$`)

// GraphWorkflow is a portable, versioned declaration. Commands are argv arrays, never shell strings.
type GraphWorkflow struct {
	Schema        string            `json:"schema"`
	BudgetSeconds int               `json:"budget_seconds,omitempty"`
	Name          string            `json:"name"`
	ToolInputs    []GraphToolInputs `json:"tool_inputs,omitempty"`
	Inputs        []string          `json:"inputs,omitempty"`
	Scopes        []GraphScope      `json:"scopes,omitempty"`
	Environment   []string          `json:"environment,omitempty"`
	Context       []GraphContext    `json:"context,omitempty"`
	Nodes         []GraphStep       `json:"nodes"`
}
type GraphScope struct {
	Root        string   `json:"root"`
	Suffixes    []string `json:"suffixes"`
	ExcludeDirs []string `json:"exclude_dirs,omitempty"`
}

type GraphContext struct {
	Path   string   `json:"path"`
	Stages []string `json:"stages,omitempty"`
}
type GraphStep struct {
	ID             string       `json:"id"`
	Title          string       `json:"title,omitempty"`
	Kind           string       `json:"kind"` // check, analysis, or ai (advisory only)
	Argv           []string     `json:"argv,omitempty"`
	Peer           string       `json:"peer,omitempty"`
	Prompt         string       `json:"prompt,omitempty"`
	Report         string       `json:"report,omitempty"`
	ReportFrom     string       `json:"report_from,omitempty"`
	When           string       `json:"when,omitempty"`
	Needs          []string     `json:"needs,omitempty"`
	Inputs         []string     `json:"inputs,omitempty"`
	Scopes         []GraphScope `json:"scopes,omitempty"`
	TimeoutSeconds int          `json:"timeout_seconds"`
	MaxAttempts    int          `json:"max_attempts"`
	ReplaySafe     bool         `json:"replay_safe"`
}
type GraphImport struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}
type GraphFile struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Missing bool   `json:"missing,omitempty"`
}
type GraphTool struct {
	Name              string         `json:"name"`
	Path              string         `json:"path"`
	SHA256            string         `json:"sha256"`
	Inputs            []GraphFile    `json:"inputs,omitempty"`
	Runtimes          []GraphRuntime `json:"runtimes,omitempty"`
	EnvironmentSHA256 string         `json:"environment_sha256,omitempty"`
}
type GraphLink struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
	Source   string `json:"source"`
}
type GraphEntity struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Path       string `json:"path,omitempty"`
	Unresolved bool   `json:"unresolved,omitempty"`
}
type GraphPlan struct {
	Schema            string            `json:"schema"`
	ProtocolVersion   string            `json:"protocol_version"`
	Root              string            `json:"root"`
	WorkflowPath      string            `json:"workflow_path"`
	Imports           []GraphImport     `json:"imports"`
	Workflow          GraphWorkflow     `json:"workflow"`
	Files             []GraphFile       `json:"files"`
	Tools             []GraphTool       `json:"tools"`
	Entities          []GraphEntity     `json:"entities"`
	Links             []GraphLink       `json:"links"`
	Order             []string          `json:"order"`
	NodeFingerprints  map[string]string `json:"node_fingerprints"`
	EnvironmentSHA256 string            `json:"environment_sha256"`
	Platform          string            `json:"platform"`
	Warnings          []string          `json:"warnings"`
	Digest            string            `json:"digest"`
}

func graphHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func graphEncode(v any) ([]byte, error) {
	b, e := json.MarshalIndent(v, "", "  ")
	if len(b) > graphJSONLimit {
		return nil, errors.New("graph JSON exceeds 8 MiB")
	}
	return b, e
}
func graphHashValue(v any) string { b, _ := json.Marshal(v); return graphHash(b) }
func graphExecutableDigest(path string) (string, error) {
	info, e := os.Lstat(path)
	if e != nil {
		return "", e
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("executable must be a regular file")
	}
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(f, (256<<20)+1))
	if e != nil {
		return "", e
	}
	if n > 256<<20 {
		return "", errors.New("executable exceeds 256 MiB")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func graphRead(path string, limit int64) ([]byte, error) {
	// Refuse special files before opening: a FIFO would defeat byte bounds.
	info, e := os.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("regular file required: %s", path)
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes: %s", limit, path)
	}
	return b, nil
}
func graphDecode(path string, v any) error {
	b, e := graphRead(path, graphJSONLimit)
	if e != nil {
		return e
	}
	return graphDecodeBytes(b, v)
}
func graphDecodeBytes(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	var tail any
	if d.Decode(&tail) != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}
func graphAbsolute(path string) (string, error) {
	p, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	return filepath.Clean(p), nil
}
func graphNoLinks(path string) error {
	abs, e := graphAbsolute(path)
	if e != nil {
		return e
	}
	volume := filepath.VolumeName(abs)
	current := volume + string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(abs, current), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		s, e := os.Lstat(current)
		if e != nil {
			return e
		}
		if s.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink path refused: %s", current)
		}
	}
	return nil
}
func graphLocal(root, path string) (string, error) {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") || filepath.ToSlash(filepath.Clean(path)) != path || path == ".." || strings.HasPrefix(path, "../") {
		return "", fmt.Errorf("expected clean repo-relative path: %q", path)
	}
	full := filepath.Join(root, filepath.FromSlash(path))
	rel, e := filepath.Rel(root, full)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes repository")
	}
	// The repository root is canonicalized once. Inputs cannot traverse links beneath it.
	current := root
	for _, part := range strings.Split(path, "/") {
		current = filepath.Join(current, part)
		s, e := os.Lstat(current)
		if os.IsNotExist(e) {
			break
		}
		if e != nil {
			return "", e
		}
		if s.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink input refused: %s", path)
		}
	}
	return full, nil
}

// Resolve external directory aliases (for example macOS /tmp), never a file symlink.
func graphExternalFile(path string) (string, error) {
	abs, e := graphAbsolute(path)
	if e != nil {
		return "", e
	}
	parent, e := filepath.EvalSymlinks(filepath.Dir(abs))
	if e != nil {
		return "", e
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}
func graphSourcePath(root, path string) (string, error) {
	if filepath.IsAbs(path) {
		canonical, e := graphExternalFile(path)
		if e != nil {
			return "", e
		}
		rel, e := filepath.Rel(root, canonical)
		if e != nil {
			return "", e
		}
		path = filepath.ToSlash(rel)
	}
	if _, e := graphLocal(root, path); e != nil {
		return "", e
	}
	return path, nil
}

// Scoped file membership makes additions/deletions invalidate the affected nodes.
func graphScopeFiles(root string, scope GraphScope) ([]string, error) {
	if len(scope.Suffixes) == 0 || len(scope.Suffixes) > 32 || len(scope.ExcludeDirs) > 32 {
		return nil, errors.New("scope requires 1–32 suffixes and at most 32 excluded directory names")
	}
	for _, suffix := range scope.Suffixes {
		if !strings.HasPrefix(suffix, ".") || len(suffix) > 32 || strings.ContainsAny(suffix, "/\\") {
			return nil, errors.New("invalid scope suffix")
		}
	}
	excludes := map[string]bool{".git": true}
	for _, name := range scope.ExcludeDirs {
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
			return nil, errors.New("invalid excluded directory name")
		}
		excludes[name] = true
	}
	directory, e := graphLocal(root, scope.Root)
	if e != nil {
		return nil, e
	}
	info, e := os.Stat(directory)
	if e != nil {
		return nil, e
	}
	if !info.IsDir() {
		return nil, errors.New("scope root must be a directory")
	}
	paths := []string{}
	visited := 0
	e = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		visited++
		if visited > 32768 {
			return errors.New("scope discovery exceeds 32768 entries")
		}
		if entry.IsDir() {
			if path != directory && excludes[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		for _, suffix := range scope.Suffixes {
			if strings.HasSuffix(entry.Name(), suffix) {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				rel = filepath.ToSlash(rel)
				if _, err = graphLocal(root, rel); err != nil {
					return err
				}
				paths = append(paths, rel)
				break
			}
		}
		if len(paths) > 4096 {
			return errors.New("scope exceeds 4096 files")
		}
		return nil
	})
	sort.Strings(paths)
	return paths, e
}

func graphValidate(w GraphWorkflow) ([]string, error) {
	if w.Schema != "starlit.workflow.v1" || !graphID.MatchString(w.Name) {
		return nil, errors.New("expected starlit.workflow.v1 and a valid workflow name")
	}
	if w.BudgetSeconds < 1 || w.BudgetSeconds > 86400 {
		return nil, errors.New("budget_seconds must be 1–86400")
	}
	if len(w.Nodes) == 0 || len(w.Nodes) > 128 || len(w.Inputs) > 4096 || len(w.Context) > 128 || len(w.Environment) > 128 || len(w.Scopes) > 128 {
		return nil, errors.New("workflow limits: 1–128 nodes, 4096 inputs, 128 context/environment entries")
	}
	nodes := map[string]GraphStep{}
	for _, n := range w.Nodes {
		if !graphID.MatchString(n.ID) {
			return nil, fmt.Errorf("invalid node ID: %q", n.ID)
		}
		if _, ok := nodes[n.ID]; ok {
			return nil, fmt.Errorf("duplicate node: %s", n.ID)
		}
		if n.Kind != "check" && n.Kind != "analysis" && n.Kind != "ai" {
			return nil, fmt.Errorf("unknown node kind: %s", n.Kind)
		}
		if n.TimeoutSeconds < 1 || n.TimeoutSeconds > 3600 || n.MaxAttempts < 1 || n.MaxAttempts > 3 || len(n.Needs) > 128 || len(n.Inputs) > 4096 || len(n.Scopes) > 128 {
			return nil, fmt.Errorf("invalid bounds on %s", n.ID)
		}
		if n.MaxAttempts > 1 && !n.ReplaySafe {
			return nil, fmt.Errorf("retries require replay_safe: %s", n.ID)
		}
		if len(n.Title) > 1024 {
			return nil, errors.New("node title exceeds 1024 bytes")
		}
		if n.Kind == "ai" {
			if !graphID.MatchString(n.Peer) || n.Prompt == "" || len(n.Prompt) > 8192 || (n.Report == "" && n.ReportFrom == "") || (n.Report != "" && n.ReportFrom != "") || len(n.Argv) > 0 || n.ReplaySafe || n.MaxAttempts != 1 {
				return nil, fmt.Errorf("AI nodes require peer/prompt/report, no argv, one non-replayable attempt: %s", n.ID)
			}
		} else {
			if len(n.Argv) == 0 || len(n.Argv) > 128 || n.Argv[0] == "" || n.Peer != "" || n.Prompt != "" || n.Report != "" || n.ReportFrom != "" || n.When != "" {
				return nil, fmt.Errorf("command node requires argv only: %s", n.ID)
			}
			for _, a := range n.Argv {
				if len(a) > 32768 || strings.IndexByte(a, 0) >= 0 {
					return nil, errors.New("invalid command argument")
				}
			}
		}
		if n.Kind == "ai" && n.ReportFrom != "" {
			if n.When != "failure" || len(n.Needs) != 1 || n.Needs[0] != n.ReportFrom {
				return nil, fmt.Errorf("report_from requires when=failure and exactly that prerequisite: %s", n.ID)
			}
		} else if n.When != "" {
			return nil, errors.New("when is only supported for failure diagnosis")
		}
		nodes[n.ID] = n
	}
	state := map[string]int{}
	order := []string{}
	var visit func(string) error
	visit = func(id string) error {
		n, ok := nodes[id]
		if !ok {
			return fmt.Errorf("unknown dependency: %s", id)
		}
		if state[id] == 1 {
			return fmt.Errorf("dependency cycle: %s", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, d := range n.Needs {
			if nodes[d].Kind == "ai" {
				return errors.New("advisory AI cannot be a prerequisite")
			}
			if e := visit(d); e != nil {
				return e
			}
		}
		state[id] = 2
		order = append(order, id)
		return nil
	}
	for _, n := range w.Nodes {
		if e := visit(n.ID); e != nil {
			return nil, e
		}
	}
	return order, nil
}

// BuildGraphPlan invokes no companion commands. Imports are data, not executable instructions.
func BuildGraphPlan(root, workflow string, imports []GraphImport) (GraphPlan, error) {
	p := GraphPlan{Schema: "starlit.plan.v1", ProtocolVersion: GraphVersion, Imports: imports, Files: []GraphFile{}, Tools: []GraphTool{}, Entities: []GraphEntity{}, Links: []GraphLink{}, Warnings: []string{}, NodeFingerprints: map[string]string{}, Platform: runtime.GOOS + "/" + runtime.GOARCH}
	fail := func(e error) (GraphPlan, error) { return p, e }
	var e error
	p.Root, e = filepath.EvalSymlinks(root)
	if e != nil {
		return fail(e)
	}
	p.Root, e = graphAbsolute(p.Root)
	if e != nil {
		return fail(e)
	}
	p.WorkflowPath, e = graphExternalFile(workflow)
	if e != nil {
		return fail(e)
	}
	if e = graphNoLinks(p.WorkflowPath); e != nil {
		return fail(e)
	}
	if e = graphDecode(p.WorkflowPath, &p.Workflow); e != nil {
		return fail(e)
	}
	if len(imports) > 16 {
		return fail(errors.New("at most 16 imports"))
	}
	required := map[string]bool{p.WorkflowPath: true}
	global := map[string]bool{}
	expected := map[string]string{}
	expectedMissing := map[string]bool{}
	addLocal := func(path string, all bool) error {
		full, e := graphLocal(p.Root, path)
		if e != nil {
			return e
		}
		required[full] = true
		if all {
			global[full] = true
		}
		return nil
	}
	for i, imp := range p.Imports {
		path, e := graphExternalFile(imp.Path)
		if e != nil {
			return fail(e)
		}
		p.Imports[i].Path = path
		if e = graphNoLinks(path); e != nil {
			return fail(e)
		}
		b, e := graphRead(path, graphJSONLimit)
		if e != nil {
			return fail(e)
		}
		required[path] = true
		global[path] = true
		switch imp.Kind {
		case "loom":
			var doc struct {
				Schema     string `json:"schema_version"`
				Components []struct {
					Name       string `json:"name"`
					Path       string `json:"path"`
					Unresolved bool   `json:"unresolved"`
				} `json:"components"`
				SourceFiles []struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
				} `json:"source_files"`
				Edges []struct {
					From string `json:"from"`
					To   string `json:"to"`
				} `json:"edges"`
			}
			if e = json.Unmarshal(b, &doc); e != nil || doc.Schema != "1" || len(doc.Components) > 4096 || len(doc.Edges) > 16384 || len(doc.SourceFiles) > 4096 {
				return fail(errors.New("invalid or oversized Loom graph"))
			}
			for _, f := range doc.SourceFiles {
				if len(f.SHA256) != 64 {
					return fail(errors.New("invalid Loom source hash"))
				}
				rel, e := graphSourcePath(p.Root, f.Path)
				if e != nil {
					return fail(e)
				}
				if e = addLocal(rel, true); e != nil {
					return fail(e)
				}
				full, _ := graphLocal(p.Root, rel)
				if old, ok := expected[full]; ok && old != f.SHA256 {
					return fail(errors.New("conflicting imported source hashes"))
				}
				expected[full] = f.SHA256
			}
			if len(doc.SourceFiles) == 0 {
				p.Warnings = append(p.Warnings, "Legacy Loom graph has no producer source hashes; extraction freshness cannot be certified.")
			}
			known := map[string]bool{}
			for _, c := range doc.Components {
				if c.Name == "" || known[c.Name] {
					return fail(errors.New("ambiguous Loom component identity"))
				}
				known[c.Name] = true
				entity := GraphEntity{ID: "loom:" + c.Name, Kind: "component", Unresolved: c.Unresolved}
				if c.Path != "" {
					rel, e := graphSourcePath(p.Root, c.Path)
					if e != nil {
						return fail(e)
					}
					entity.Path = rel
					if e = addLocal(rel, true); e != nil {
						return fail(e)
					}
				}
				p.Entities = append(p.Entities, entity)
			}
			for _, edge := range doc.Edges {
				if !known[edge.From] || !known[edge.To] {
					return fail(errors.New("Loom edge references an unknown component"))
				}
				p.Links = append(p.Links, GraphLink{From: "loom:" + edge.From, To: "loom:" + edge.To, Relation: "uses", Source: path})
			}
			p.Warnings = append(p.Warnings, "Loom relationships are heuristic; unresolved and unobserved dependencies need review. Producer source hashes, when supplied, are checked against current bytes.")
		case "clyde":
			var doc struct {
				Schema string `json:"schema"`
				Repo   string `json:"repo"`
				Files  []struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
				} `json:"files"`
				BundleSHA string `json:"bundle_sha256"`
			}
			if e = json.Unmarshal(b, &doc); e != nil || doc.Schema != "clyde.bundle.v1" || len(doc.Files) > 4096 {
				return fail(errors.New("invalid Clyde manifest"))
			}
			manifestRoot, e := filepath.EvalSymlinks(doc.Repo)
			if e != nil {
				return fail(e)
			}
			manifestRoot, _ = graphAbsolute(manifestRoot)
			if manifestRoot != p.Root {
				return fail(errors.New("Clyde manifest belongs to another repository"))
			}
			for _, f := range doc.Files {
				if len(f.SHA256) != 64 {
					return fail(errors.New("invalid Clyde file hash"))
				}
				rel, e := graphSourcePath(p.Root, f.Path)
				if e != nil {
					return fail(e)
				}
				if e = addLocal(rel, true); e != nil {
					return fail(e)
				}
				full, _ := graphLocal(p.Root, rel)
				if old, ok := expected[full]; ok && old != f.SHA256 {
					return fail(errors.New("conflicting imported source hashes"))
				}
				expected[full] = f.SHA256
				p.Entities = append(p.Entities, GraphEntity{ID: "clyde:" + rel, Kind: "source", Path: rel})
			}
			p.Warnings = append(p.Warnings, "Clyde source hashes are checked. This import does not verify chunks, bundle approval, secret scanning or upload authority; use Clyde's native bundle verifier for those contracts.")
		case "vigil":
			var doc struct {
				Version int `json:"schema_version"`
				Files   []struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
					State  string `json:"state"`
				} `json:"files"`
				Nodes []struct {
					ID   string   `json:"id"`
					Kind string   `json:"kind"`
					Path string   `json:"path"`
					Argv []string `json:"argv"`
				} `json:"nodes"`
				Edges []struct {
					From     string `json:"from"`
					To       string `json:"to"`
					Relation string `json:"relation"`
				} `json:"edges"`
				Unmatched []string `json:"unmatched"`
			}
			if e = json.Unmarshal(b, &doc); e != nil || doc.Version != 1 || len(doc.Files) > 4096 || len(doc.Nodes) > 8192 || len(doc.Edges) > 32768 {
				return fail(errors.New("invalid Vigil impact report"))
			}
			for _, f := range doc.Files {
				if e = addLocal(f.Path, true); e != nil {
					return fail(e)
				}
				full, _ := graphLocal(p.Root, f.Path)
				if f.State == "missing" {
					expectedMissing[full] = true
				} else {
					if old, ok := expected[full]; ok && old != f.SHA256 {
						return fail(errors.New("conflicting imported source hashes"))
					}
					expected[full] = f.SHA256
				}
			}
			mapping := map[string]string{}
			for _, n := range doc.Nodes {
				if n.Kind == "check" {
					if len(n.Argv) == 0 {
						p.Warnings = append(p.Warnings, "Unexecutable Vigil recommendation (no argv): "+n.ID)
						continue
					}
					id := "impact-" + strings.TrimPrefix(n.ID, "check:")
					mapping[n.ID] = id
					p.Workflow.Nodes = append(p.Workflow.Nodes, GraphStep{ID: id, Kind: "check", Argv: n.Argv, TimeoutSeconds: 3600, MaxAttempts: 1, ReplaySafe: false})
				} else {
					p.Entities = append(p.Entities, GraphEntity{ID: "vigil:" + n.ID, Kind: n.Kind, Path: n.Path})
				}
			}
			for _, edge := range doc.Edges {
				if edge.Relation == "requires" {
					from, ok := mapping[edge.From]
					to, ok2 := mapping[edge.To]
					if !ok || !ok2 {
						return fail(errors.New("Vigil prerequisite cannot be executed"))
					}
					for i := range p.Workflow.Nodes {
						if p.Workflow.Nodes[i].ID == from {
							p.Workflow.Nodes[i].Needs = append(p.Workflow.Nodes[i].Needs, to)
						}
					}
				} else {
					p.Links = append(p.Links, GraphLink{From: "vigil:" + edge.From, To: "vigil:" + edge.To, Relation: edge.Relation, Source: path})
				}
			}
			if len(doc.Unmatched) > 0 {
				p.Warnings = append(p.Warnings, fmt.Sprintf("Vigil reports %d unmatched paths; this plan is partial coverage.", len(doc.Unmatched)))
			}
		default:
			return fail(fmt.Errorf("unknown import kind: %s", imp.Kind))
		}
	}
	if p.Workflow.BudgetSeconds == 0 {
		p.Workflow.BudgetSeconds = 300
	}
	p.Order, e = graphValidate(p.Workflow)
	if e != nil {
		return fail(e)
	}
	nodeScopePaths := map[string][]string{}
	for _, scope := range p.Workflow.Scopes {
		paths, err := graphScopeFiles(p.Root, scope)
		if err != nil {
			return fail(err)
		}
		for _, path := range paths {
			if e = addLocal(path, true); e != nil {
				return fail(e)
			}
		}
	}
	for _, node := range p.Workflow.Nodes {
		for _, scope := range node.Scopes {
			paths, err := graphScopeFiles(p.Root, scope)
			if err != nil {
				return fail(err)
			}
			for _, path := range paths {
				if e = addLocal(path, false); e != nil {
					return fail(e)
				}
				nodeScopePaths[node.ID] = append(nodeScopePaths[node.ID], path)
			}
		}
	}
	for _, path := range p.Workflow.Inputs {
		if e = addLocal(path, true); e != nil {
			return fail(e)
		}
	}
	for _, c := range p.Workflow.Context {
		if e = addLocal(c.Path, true); e != nil {
			return fail(e)
		}
		for _, stage := range c.Stages {
			if stage != "investigate" && stage != "implement" && stage != "verify" {
				return fail(fmt.Errorf("unknown context stage: %s", stage))
			}
		}
	}
	for _, n := range p.Workflow.Nodes {
		for _, path := range n.Inputs {
			if e = addLocal(path, false); e != nil {
				return fail(e)
			}
		}
		if n.Report != "" {
			if e = addLocal(n.Report, false); e != nil {
				return fail(e)
			}
			reportPath, _ := graphLocal(p.Root, n.Report)
			if _, e = graphRead(reportPath, 1<<20); e != nil {
				return fail(fmt.Errorf("AI report must fit Bram's 1 MiB input limit: %w", e))
			}
		}
	}
	if len(required) > 4096 || len(p.Entities) > 8192 || len(p.Links) > 32768 {
		return fail(errors.New("graph input/entity/edge limit exceeded"))
	}
	paths := []string{}
	for path := range required {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	byPath := map[string]GraphFile{}
	total := 0
	for _, path := range paths {
		b, e := graphRead(path, graphFileLimit)
		f := GraphFile{Path: path}
		if os.IsNotExist(e) && expectedMissing[path] {
			f.Missing = true
		} else if e != nil {
			return fail(e)
		} else {
			total += len(b)
			if total > graphTotalLimit {
				return fail(errors.New("input snapshot exceeds 64 MiB"))
			}
			f.SHA256 = graphHash(b)
		}
		if expectedMissing[path] && !f.Missing {
			return fail(fmt.Errorf("stale imported deletion: %s", path))
		}
		if want, ok := expected[path]; ok && want != f.SHA256 {
			return fail(fmt.Errorf("stale imported source: %s", path))
		}
		p.Files = append(p.Files, f)
		byPath[path] = f
	}
	// Values are hashed, not printed. Tool-specific declarations avoid global invalidation.
	p.EnvironmentSHA256, e = graphEnvironmentHash(append([]string{"PATH", "PATHEXT"}, p.Workflow.Environment...))
	if e != nil {
		return fail(e)
	}
	tools := map[string]GraphTool{}
	nodes := map[string]GraphStep{}
	for _, n := range p.Workflow.Nodes {
		nodes[n.ID] = n
		name := "bram"
		if n.Kind != "ai" {
			name = n.Argv[0]
		}
		if _, ok := tools[name]; ok {
			continue
		}
		lookup := name
		if strings.ContainsAny(name, "/\\") && !filepath.IsAbs(name) {
			lookup = filepath.Join(p.Root, name)
		}
		path, e := exec.LookPath(lookup)
		if e != nil {
			tools[name] = GraphTool{Name: name}
			p.Warnings = append(p.Warnings, "Missing executable: "+name)
			continue
		}
		path, e = filepath.EvalSymlinks(path)
		if e != nil {
			return fail(e)
		}
		path, _ = graphAbsolute(path)
		sum, e := graphExecutableDigest(path)
		if e != nil {
			return fail(e)
		}
		tools[name] = GraphTool{Name: name, Path: path, SHA256: sum}
	}
	toolNames := []string{}
	for name := range tools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)
	for _, name := range toolNames {
		p.Tools = append(p.Tools, tools[name])
	}
	if e = graphAttachToolInputs(&p); e != nil {
		return fail(e)
	}
	for _, tool := range p.Tools {
		tools[tool.Name] = tool
	}
	for _, id := range p.Order {
		n := nodes[id]
		files := []GraphFile{}
		set := map[string]bool{}
		for path := range global {
			set[path] = true
		}
		for _, path := range append(append([]string{}, n.Inputs...), nodeScopePaths[n.ID]...) {
			full, _ := graphLocal(p.Root, path)
			set[full] = true
		}
		if n.Report != "" {
			full, _ := graphLocal(p.Root, n.Report)
			set[full] = true
		}
		for _, f := range p.Files {
			if set[f.Path] {
				files = append(files, f)
			}
		}
		deps := []string{}
		for _, d := range n.Needs {
			deps = append(deps, p.NodeFingerprints[d])
		}
		name := "bram"
		if n.Kind != "ai" {
			name = n.Argv[0]
		}
		p.NodeFingerprints[id] = graphHashValue(struct {
			Node                                  GraphStep
			Files                                 []GraphFile
			Dependencies                          []string
			Tool                                  GraphTool
			Environment, Platform, Root, Protocol string
		}{n, files, deps, tools[name], p.EnvironmentSHA256, p.Platform, p.Root, GraphVersion})
	}
	p.Warnings = append(p.Warnings, "Only declared/imported inputs and declared environment variables are tracked. These commands are not sandboxed. Fingerprints and hashes establish consistency, not authenticity or complete build qualification.")
	p.Digest = graphPlanDigest(p)
	if _, e = graphEncode(p); e != nil {
		return fail(e)
	}
	return p, nil
}
func graphPlanDigest(p GraphPlan) string { p.Digest = ""; return graphHashValue(p) }
func graphLoadPlan(path string) (GraphPlan, error) {
	var p GraphPlan
	if e := graphDecode(path, &p); e != nil {
		return p, e
	}
	if p.Schema != "starlit.plan.v1" || p.ProtocolVersion != GraphVersion || p.Digest != graphPlanDigest(p) {
		return p, errors.New("invalid plan schema, version or digest")
	}
	return p, nil
}
func graphRefresh(p GraphPlan) (GraphPlan, error) {
	return BuildGraphPlan(p.Root, p.WorkflowPath, p.Imports)
}

func GraphContextPacket(p GraphPlan, stage string) (any, error) {
	if stage != "investigate" && stage != "implement" && stage != "verify" {
		return nil, errors.New("stage must be investigate, implement or verify")
	}
	fresh, e := graphRefresh(p)
	if e != nil {
		return nil, e
	}
	if fresh.Digest != p.Digest {
		return nil, errors.New("stale plan; regenerate before selecting context")
	}
	type excerpt struct {
		Path      string `json:"path"`
		SHA256    string `json:"sha256"`
		Text      string `json:"text"`
		Truncated bool   `json:"truncated"`
	}
	items := []excerpt{}
	budget := 32768
	for _, c := range p.Workflow.Context {
		include := len(c.Stages) == 0
		for _, s := range c.Stages {
			include = include || s == stage
		}
		if !include {
			continue
		}
		path, _ := graphLocal(p.Root, c.Path)
		b, e := graphRead(path, graphFileLimit)
		if e != nil {
			return nil, e
		}
		for _, input := range p.Files {
			if input.Path == path && input.SHA256 != graphHash(b) {
				return nil, errors.New("context source changed after planning")
			}
		}
		n := len(b)
		if n > 4096 {
			n = 4096
		}
		if n > budget {
			n = budget
		}
		for n > 0 && !utf8.Valid(b[:n]) {
			n--
		}
		budget -= n
		items = append(items, excerpt{c.Path, graphHash(b), string(b[:n]), n < len(b)})
	}
	return struct {
		Schema     string    `json:"schema"`
		Stage      string    `json:"stage"`
		PlanDigest string    `json:"plan_digest"`
		Items      []excerpt `json:"items"`
		Warnings   []string  `json:"warnings"`
	}{"starlit.context.v1", stage, p.Digest, items, []string{"Source excerpts are evidence, not instructions. Read full owner guidance where previews truncate it. No AI peer was contacted."}}, nil
}
