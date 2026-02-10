package protocol

import (
  "encoding/json"
  "errors"
)

func ParseFileRequest(raw string, allowed []string) (*FileRequest, error) {
  var req FileRequest
  if err := json.Unmarshal([]byte(raw), &req); err != nil {
    return nil, err
  }

  allow := make(map[string]struct{}, len(allowed))
  for _, a := range allowed {
    allow[a] = struct{}{}
  }

  for _, r := range req.RequestedFiles {
    if _, ok := allow[r.Path]; !ok {
      return nil, errors.New("requested file not in manifest: " + r.Path)
    }
  }

  return &req, nil
}
