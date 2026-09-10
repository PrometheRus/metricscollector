package agent

import (
	"bytes"
	"compress/gzip"
	"fmt"
)

// Compress returns a gzip-compressed copy of data using best compression.
func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer

	// BestCompression is a package constant, so NewWriterLevel never errors here.
	w, _ := gzip.NewWriterLevel(&b, gzip.BestCompression)

	_, err := w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write data to compression buffer: %v", err)
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to compress data: %v", err)
	}

	return b.Bytes(), nil
}
