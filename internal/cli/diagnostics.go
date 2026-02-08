package cli

import "fmt"

func DumpSystemPrompt() {
  if runtime == nil || runtime.SystemPrompt == nil {
    fmt.Println("system prompt not initialized")
    return
  }

  fmt.Println("----- BEGIN SYSTEM PROMPT -----")
  fmt.Println(runtime.SystemPrompt.Raw())
  fmt.Println("----- END SYSTEM PROMPT -----")
}
