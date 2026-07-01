package resume

import (
	"bytes"
	"fmt"

	docx "github.com/lukasjarosch/go-docx"
)

// Fill opens the base template at templatePath, replaces every placeholder
// key in values (see placeholders.go), and returns the resulting .docx
// bytes. The base template is never modified — only a copy is written.
func Fill(templatePath string, values map[string]string) ([]byte, error) {
	doc, err := docx.Open(templatePath)
	if err != nil {
		return nil, fmt.Errorf("open template %s: %w", templatePath, err)
	}
	defer doc.Close()

	placeholders := make(docx.PlaceholderMap, len(values))
	for k, v := range values {
		placeholders[k] = v
	}

	if err := doc.ReplaceAll(placeholders); err != nil {
		return nil, fmt.Errorf("replace placeholders: %w", err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		return nil, fmt.Errorf("write filled docx: %w", err)
	}
	return buf.Bytes(), nil
}
