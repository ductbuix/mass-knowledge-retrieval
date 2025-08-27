package index

import (
	"io"
	"os"
)

// ReadPreview returns at most n characters from the beginning of the given
// file. It is used to display a short snippet of a chunk when printing
// search results. If the file is shorter than n, the entire content is
// returned. Errors are silently swallowed and result in an empty string.
func ReadPreview(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, n)
	// read up to n bytes
	m, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return ""
	}
	return string(buf[:m])
}