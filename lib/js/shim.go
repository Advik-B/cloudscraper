package js

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/url"
	"sync"
	"text/template"
)

//go:embed setup.js
var shimTemplate string

var (
	jsTpl  *template.Template
	once   sync.Once
	tplErr error
)

// JSTemplateData holds the dynamic data to be injected into the JS shim.
type JSTemplateData struct {
	Href     string
	Scheme   string
	Host     string
	Hostname string
	Port     string
	Path     string
	RawQuery string
	Fragment string
}

// GenerateShim populates the JavaScript shim template with dynamic data from the given URL.
func GenerateShim(pageURL *url.URL) (string, error) {
	once.Do(func() {
		jsTpl, tplErr = template.New("shim").Parse(shimTemplate)
	})
	if tplErr != nil {
		return "", fmt.Errorf("failed to parse JS shim template: %w", tplErr)
	}

	data := JSTemplateData{
		Href:     pageURL.String(),
		Scheme:   pageURL.Scheme,
		Host:     pageURL.Host,
		Hostname: pageURL.Hostname(),
		Port:     pageURL.Port(),
		Path:     pageURL.Path,
		RawQuery: pageURL.RawQuery,
		Fragment: pageURL.Fragment,
	}

	var buf bytes.Buffer
	if err := jsTpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute JS shim template: %w", err)
	}

	return buf.String(), nil
}
