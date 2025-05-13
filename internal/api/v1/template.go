package v1

import (
	"bytes"
	"fmt"
	"text/template"
)

// executeTemplate demonstrates basic usage of text/template.
// It takes a template string and data, then returns the executed result or an error.
func executeTemplate(templateStr string, data interface{}) (string, error) {
	tmpl, err := template.New("apiTemplate").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// Example usage (can be integrated into handlers later):
/*
func someHandler(w http.ResponseWriter, r *http.Request) {
	templateString := "Hello, {{.Name}}! Your ID is {{.ID}}."
	data := struct {
		Name string
		ID   int
	}{
		Name: "User",
		ID:   123,
	}

	result, err := executeTemplate(templateString, data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, result)
}
*/
