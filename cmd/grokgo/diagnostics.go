package main

import (
  "encoding/json"
  "fmt"
  "os"

  "grokgo/internal/diagnostics/safety"
)

func runDiagnostics(jsonOut bool) {
  results, safe := safety.Run()

  if jsonOut {
    enc := json.NewEncoder(os.Stdout)
    if !safe {
      enc.Encode(map[string]any{
        "verdict": "unsafe",
        "exit_code": 3,
        "violations": results,
      })
      os.Exit(3)
    }

    enc.Encode(map[string]any{
      "verdict": "safe",
      "exit_code": 0,
    })
    return
  }

  if !safe {
    fmt.Println("🔴 Safety invariant violation")
    os.Exit(3)
  }

  fmt.Println("🟢 Safety invariants passed")
}
