package cli

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
)

type Permissions struct {
	FSRead  bool
	FSWrite bool
}

func RequestFSReadPermission(cwd string) bool {
	items := []string{
		"Allow read-only access (this session)",
		"Deny",
		"Abort request",
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf(
			"Goshi requires permission to read files in:\n  %s\n\nWhat would you like to do?",
			cwd,
		),
		Items: items,
	}

	i, _, err := prompt.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "permission prompt cancelled")
		return false
	}

	switch i {
	case 0:
		return true
	case 1:
		return false
	default:
		return false
	}
}

func RequestFSWritePermission(cwd string) bool {
	items := []string{
		"Allow write access (this session)",
		"Deny",
		"Abort request",
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf(
			"Goshi requests permission to write files in:\n  %s\n\nWhat would you like to do?",
			cwd,
		),
		Items: items,
	}

	i, _, err := prompt.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "permission prompt cancelled")
		return false
	}

	switch i {
	case 0:
		return true
	case 1:
		return false
	default:
		return false
	}
}
