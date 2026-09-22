package server

import "github.com/hkjang/madi/internal/extract"

// RunAttachmentWorker is called before the service reads its four bootstrap
// variables. Only fixed Poppler/Tesseract operations are accepted by the child.
func RunAttachmentWorker(args []string) (bool, int) { return extract.RunWorker(args) }
