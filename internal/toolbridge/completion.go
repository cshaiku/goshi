// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"fmt"
	"io"
	"strings"
)

// ToolCompletions is a pure query: it never inspects files or invokes companions.
// Words are relative to the tools namespace and include the current partial word.
func ToolCompletions(words []string) []string {
	if len(words) == 0 {
		words = []string{""}
	}
	prefix := words[len(words)-1]
	candidates := []string{"doctor", "identity", "plan", "run", "graph", "completion"}
	if len(words) > 1 {
		if words[0] != "graph" {
			return nil
		}
		candidates = []string{"capabilities", "schema", "plan", "explain", "context", "run", "resume", "verify", "help"}
		if len(words) > 2 {
			previous := words[len(words)-2]
			switch previous {
			case "--kind":
				candidates = []string{"workflow", "plan", "run", "verification"}
			case "--stage":
				candidates = []string{"investigate", "implement", "verify"}
			case "--format":
				candidates = []string{"json", "text", "mermaid"}
			default:
				switch words[1] {
				case "schema":
					candidates = []string{"--kind"}
				case "plan":
					candidates = []string{"--repo", "--workflow", "--loom", "--clyde-manifest", "--impact"}
				case "explain":
					candidates = []string{"--plan", "--run-dir", "--format"}
				case "context":
					candidates = []string{"--plan", "--stage"}
				case "run", "resume":
					candidates = []string{"--plan", "--run-dir", "--approve-digest", "--allow-ai"}
				case "verify":
					candidates = []string{"--run-dir"}
				default:
					return nil
				}
			}
		}
	}
	result := []string{}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			result = append(result, candidate)
		}
	}
	return result
}

// toolCompletion emits an opt-in shell hook confined to the tools namespace.
func toolCompletion(shell, caller string, out io.Writer) error {
	var script string
	switch shell {
	case "bash":
		script = `_STARLIT_tools_completion() {
  [[ "${COMP_WORDS[1]}" == tools ]] || return 0
  local IFS=$'\n'
  COMPREPLY=($("${COMP_WORDS[0]}" tools complete -- "${COMP_WORDS[@]:2:$COMP_CWORD-1}"))
}
complete -o default -F _STARLIT_tools_completion TOOL
`
	case "zsh":
		script = `#compdef TOOL
_STARLIT_tools_completion() {
 [[ "${words[2]}" == tools ]] || return 1
 local -a suggestions
 suggestions=("${(@f)$("${words[1]}" tools complete -- "${words[@]:2}")}")
 compadd -- "${suggestions[@]}"
}
compdef _STARLIT_tools_completion TOOL
`
	case "fish":
		script = `function _STARLIT_tools_completion
 set -l words (commandline -opc)
 if test (count $words) -ge 2; and test "$words[2]" = tools
  command TOOL tools complete -- $words[3..-1] (commandline -ct)
 end
end
complete -c TOOL -n '__fish_seen_subcommand_from tools' -a '(_STARLIT_tools_completion)'
`
	case "powershell":
		script = `Register-ArgumentCompleter -Native -CommandName TOOL -ScriptBlock {
 param($wordToComplete, $commandAst, $cursorPosition)
 $parts = @($commandAst.CommandElements | ForEach-Object { $_.Extent.Text })
 if ($parts.Count -lt 2 -or $parts[1] -ne 'tools') { return }
 $argsForTools = @()
 if ($parts.Count -gt 2) { $argsForTools = @($parts[2..($parts.Count-1)]) }
 if ($wordToComplete -eq '') { $argsForTools += '' }
 & TOOL tools complete -- @argsForTools | ForEach-Object {
  [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
 }
}
`
	default:
		return fmt.Errorf("completion shell must be bash, zsh, fish or powershell")
	}
	_, err := io.WriteString(out, strings.ReplaceAll(strings.ReplaceAll(script, "STARLIT", caller), "TOOL", caller))
	return err
}
