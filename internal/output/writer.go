package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vettcode/scanner/pkg/models"
)

// WriteScanResult writes the ScanResult to the specified path as pretty-printed JSON.
// Uses atomic write (temp file + rename) to prevent partial files on failure.
func WriteScanResult(result *models.ScanResult, outputPath string) error {
	data, err := marshalPretty(result)
	if err != nil {
		return fmt.Errorf("marshal scan result: %w", err)
	}
	return atomicWrite(outputPath, data)
}

// marshalPretty produces indented JSON with SetEscapeHTML(false).
func marshalPretty(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// atomicWrite writes data to a temp file in the same directory, then renames.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vettcode-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
