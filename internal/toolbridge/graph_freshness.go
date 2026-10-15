// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// GraphToolInputs describes observable configuration and runtime identities, not
// a promise of hermetic execution. Files can be repo-relative or explicit absolute paths.
type GraphToolInputs struct {
	Name          string   `json:"name"`
	Files         []string `json:"files,omitempty"`
	OptionalFiles []string `json:"optional_files,omitempty"`
	Runtimes      []string `json:"runtimes,omitempty"`
	Environment   []string `json:"environment,omitempty"`
}
type GraphRuntime struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func graphEnvironmentHash(names []string) (string, error) {
	unique := map[string]bool{}
	for _, name := range names {
		if name == "" || strings.ContainsAny(name, "=\x00") {
			return "", errors.New("invalid environment name")
		}
		unique[name] = true
	}
	names = names[:0]
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	values := []string{}
	for _, name := range names {
		value, present := os.LookupEnv(name)
		values = append(values, fmt.Sprintf("%s:%t:%s", name, present, value))
	}
	return graphHashValue(values), nil
}
func graphResolveExecutable(root, name string) (GraphRuntime, error) {
	r := GraphRuntime{Name: name}
	lookup := name
	if strings.ContainsAny(name, "/\\") && !filepath.IsAbs(name) {
		lookup = filepath.Join(root, name)
	}
	path, e := exec.LookPath(lookup)
	if e != nil {
		return r, fmt.Errorf("missing runtime executable %s: %w", name, e)
	}
	path, e = filepath.EvalSymlinks(path)
	if e != nil {
		return r, e
	}
	r.Path, _ = graphAbsolute(path)
	r.SHA256, e = graphExecutableDigest(r.Path)
	return r, e
}
func graphToolFile(root, path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		path = filepath.Join(home, filepath.FromSlash(path[2:]))
	}
	if !filepath.IsAbs(path) {
		return graphLocal(root, path)
	}
	abs, e := graphAbsolute(path)
	if e != nil {
		return "", e
	}
	// Canonicalize existing directory aliases, preserving missing optional suffixes.
	parent := filepath.Dir(abs)
	suffix := []string{filepath.Base(abs)}
	for {
		if _, e = os.Lstat(parent); e == nil {
			break
		}
		if !os.IsNotExist(e) {
			return "", e
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", errors.New("no existing input parent")
		}
		suffix = append([]string{filepath.Base(parent)}, suffix...)
		parent = next
	}
	parent, e = filepath.EvalSymlinks(parent)
	if e != nil {
		return "", e
	}
	return filepath.Join(append([]string{parent}, suffix...)...), nil
}
func graphAttachToolInputs(p *GraphPlan) error {
	if len(p.Workflow.ToolInputs) > 128 {
		return errors.New("at most 128 tool input declarations")
	}
	seen := map[string]bool{}
	total := 0
	for _, f := range p.Files {
		if !f.Missing {
			if info, e := os.Stat(f.Path); e == nil {
				total += int(info.Size())
			}
		}
	}
	tracked := map[string]GraphFile{}
	for _, f := range p.Files {
		tracked[f.Path] = f
	}
	for _, spec := range p.Workflow.ToolInputs {
		if spec.Name == "" || seen[spec.Name] || len(spec.Files)+len(spec.OptionalFiles) > 128 || len(spec.Runtimes) > 16 || len(spec.Environment) > 128 {
			return errors.New("invalid or duplicate tool input declaration")
		}
		seen[spec.Name] = true
		index := -1
		for i, t := range p.Tools {
			if t.Name == spec.Name {
				index = i
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("tool inputs reference unused executable: %s", spec.Name)
		}
		tool := &p.Tools[index]
		files := map[string]GraphFile{}
		for group, paths := range [][]string{spec.Files, spec.OptionalFiles} {
			for _, path := range paths {
				full, e := graphToolFile(p.Root, path)
				if e != nil {
					return e
				}
				b, e := graphRead(full, graphFileLimit)
				f := GraphFile{Path: full}
				if os.IsNotExist(e) && group == 1 {
					f.Missing = true
				} else if e != nil {
					return e
				} else {
					f.SHA256 = graphHash(b)
				}
				if prior, ok := tracked[full]; ok {
					if prior != f {
						return errors.New("tool input changed while planning")
					}
				} else {
					total += len(b)
					if total > graphTotalLimit {
						return errors.New("input snapshot exceeds 64 MiB")
					}
					tracked[full] = f
					p.Files = append(p.Files, f)
				}
				if old, ok := files[full]; ok && old != f {
					return errors.New("conflicting tool file declarations")
				}
				files[full] = f
			}
		}
		for _, f := range files {
			tool.Inputs = append(tool.Inputs, f)
		}
		sort.Slice(tool.Inputs, func(i, j int) bool { return tool.Inputs[i].Path < tool.Inputs[j].Path })
		var e error
		tool.EnvironmentSHA256, e = graphEnvironmentHash(append([]string{}, spec.Environment...))
		if e != nil {
			return e
		}
		runtimeNames := map[string]bool{}
		for _, name := range spec.Runtimes {
			if name == "" || runtimeNames[name] {
				return errors.New("invalid or duplicate runtime name")
			}
			runtimeNames[name] = true
			r, e := graphResolveExecutable(p.Root, name)
			if e != nil {
				return e
			}
			tool.Runtimes = append(tool.Runtimes, r)
		}
		sort.Slice(tool.Runtimes, func(i, j int) bool { return tool.Runtimes[i].Name < tool.Runtimes[j].Name })
	}
	if len(p.Files) > 4096 {
		return errors.New("too many tool/snapshot input paths")
	}
	sort.Slice(p.Files, func(i, j int) bool { return p.Files[i].Path < p.Files[j].Path })
	return nil
}
