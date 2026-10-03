// Package webui owns Spectre's shared browser presentation and theme behavior.
package webui

import (
	_ "embed"
	"html/template"
)

// Mode identifies the capability displayed in the application header.
type Mode string

const (
	// Specs identifies the Markdown specification viewer.
	Specs Mode = "Specs"
	// Audit identifies an audit report, including standalone exports.
	Audit Mode = "Audit"
)

// Header contains the current mode and document name. Home may be empty for
// exported pages that have no local application route.
type Header struct {
	Mode Mode
	Name string
	Home string
}

//go:embed assets/style.css
var styles string

//go:embed assets/theme.js
var themeScript string

//go:embed assets/header.html
var headerTemplate string

// Styles returns the embedded application CSS for inline or HTTP delivery.
// Mode styles follow the shared styles so their overrides retain precedence.
func Styles(modeStyles []byte) template.CSS { return template.CSS(styles + "\n" + string(modeStyles)) }

// ThemeScript returns the embedded theme initializer and interactive controls.
func ThemeScript() template.JS { return template.JS(themeScript) }

// Templates adds the shared application header to the caller's template set.
func Templates(t *template.Template) (*template.Template, error) { return t.Parse(headerTemplate) }
