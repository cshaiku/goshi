package app

import (
	"github.com/cshaiku/goshi/internal/actions/runtime"
	"github.com/cshaiku/goshi/internal/fs"
)

// ActionService is a minimal façade over the action dispatcher.
type ActionService struct {
	dispatcher *runtime.Dispatcher
}

func (s *ActionService) Dispatcher() *runtime.Dispatcher {
  return s.dispatcher
}

// NewActionService creates an ActionService scoped to a root directory.
func NewActionService(root string) (*ActionService, error) {
	guard, err := fs.NewGuard(root)
	if err != nil {
		return nil, err
	}

	dispatcher := runtime.NewDispatcher(guard)

	return &ActionService{
		dispatcher: dispatcher,
	}, nil
}

// RunAction executes a named action with inputs.
// This is intentionally thin and explicit.
func (s *ActionService) RunAction(
	action string,
	input map[string]any,
) (map[string]any, error) {
	return s.dispatcher.Dispatch(action, input)
}
