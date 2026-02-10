package runtime

import (
	"errors"
	"os"
	"time"

	"github.com/cshaiku/goshi/internal/fs"
)

// ActionInput is a generic action invocation payload.
// This mirrors the shape defined in actions.yaml.
type ActionInput map[string]any

// ActionOutput is a generic action result payload.
type ActionOutput map[string]any

var (
	ErrUnknownAction = errors.New("unknown action")
	ErrInvalidInput  = errors.New("invalid action input")
)

// Dispatcher routes actions to concrete implementations.
type Dispatcher struct {
	guard *fs.Guard
}

// NewDispatcher creates a dispatcher scoped to a filesystem guard.
func NewDispatcher(guard *fs.Guard) *Dispatcher {
	return &Dispatcher{guard: guard}
}

// Dispatch executes a named action with validated inputs.
func (d *Dispatcher) Dispatch(action string, in ActionInput) (ActionOutput, error) {
	switch action {

	case "fs.read":
		path, ok := in["path"].(string)
		if !ok {
			return nil, ErrInvalidInput
		}

		res, err := fs.Read(d.guard, path)
		if err != nil {
			return nil, err
		}

		return ActionOutput{
			"path":    res.Path,
			"content": res.Content,
			"size":    res.Size,
		}, nil

	case "fs.list":
		path, ok := in["path"].(string)
		if !ok {
			return nil, ErrInvalidInput
		}

		res, err := fs.List(d.guard, path)
		if err != nil {
			return nil, err
		}

		entries := make([]ActionOutput, 0, len(res.Entries))
		for _, e := range res.Entries {
			entries = append(entries, ActionOutput{
				"name":   e.Name,
				"path":   e.Path,
				"is_dir": e.IsDir,
				"size":   e.Size,
			})
		}

		return ActionOutput{
			"path":    res.Path,
			"entries": entries,
		}, nil

	case "fs.write":
		path, ok1 := in["path"].(string)
		content, ok2 := in["content"].(string)
		if !ok1 || !ok2 {
			return nil, ErrInvalidInput
		}

		resolved, err := d.guard.Resolve(path)
		if err != nil {
			return nil, err
		}

		var (
			isNew    = true
			baseHash = ""
		)

		if data, err := os.ReadFile(resolved); err == nil {
			isNew = false
			baseHash = fs.ComputeHash(data)
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		contentHash := fs.ComputeHash([]byte(content))

		p := fs.Proposal{
			ID:          fs.ProposalID(resolved, isNew, baseHash, contentHash),
			Path:        resolved,
			IsNewFile:   isNew,
			BaseHash:    baseHash,
			ContentHash: contentHash,
			GeneratedAt: time.Now().UTC(),
		}

		if err := fs.SaveProposal(p); err != nil {
			return nil, err
		}

		return ActionOutput{
			"id":           p.ID,
			"path":         p.Path,
			"is_new_file":  p.IsNewFile,
			"base_hash":    p.BaseHash,
			"content_hash": p.ContentHash,
			"generated_at": p.GeneratedAt,
		}, nil

	case "fs.list-recursive":
		path, ok := in["path"].(string)
		if !ok {
			return nil, ErrInvalidInput
		}

		res, err := fs.ListRecursive(d.guard, path)
		if err != nil {
			return nil, err
		}

		return ActionOutput{
			"path":  res.Path,
			"files": res.Files,
			"count": res.Count,
		}, nil

	default:
		return nil, ErrUnknownAction
	}
}
