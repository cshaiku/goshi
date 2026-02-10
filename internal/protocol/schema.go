package protocol

type FileRequest struct {
  RequestedFiles []RequestedFile `json:"requested_files"`
}

type RequestedFile struct {
  Path   string `json:"path"`
  Reason string `json:"reason"`
}
