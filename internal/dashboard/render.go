package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"io"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// stepNode is the per-stepper-node view model: its CSS state class, the glyph
// shown inside the dot, the step label, and whether a (possibly filled)
// connector follows it. It mirrors app.js renderStepper exactly.
type stepNode struct {
	Class    string // "" | "done" | "cur" | "err"
	Glyph    string // "" | "✓" | "✕"
	Label    string
	HasConn  bool
	ConnFill bool
}

// recoverResult is the view model for the toast template.
type recoverResult struct {
	Message string
}

// Renderer renders named dashboard regions to HTML from a DashboardState.
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer parses the embedded templates and wires the helper FuncMap.
func NewRenderer() (*Renderer, error) {
	tmpl := template.New("dashboard").Funcs(funcMap())
	tmpl, err := tmpl.ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse dashboard templates: %w", err)
	}
	return &Renderer{tmpl: tmpl}, nil
}

// Region renders a single named region (dep, kpis, orders, versions, controls)
// to w using the given state.
func (r *Renderer) Region(w io.Writer, name string, state DashboardState) error {
	if err := r.tmpl.ExecuteTemplate(w, name, state); err != nil {
		return fmt.Errorf("render region %q: %w", name, err)
	}
	return nil
}

// Toast renders the toast fragment with the given message.
func (r *Renderer) Toast(w io.Writer, message string) error {
	if err := r.tmpl.ExecuteTemplate(w, "toast", recoverResult{Message: message}); err != nil {
		return fmt.Errorf("render toast: %w", err)
	}
	return nil
}

// funcMap ports the app.js rendering helpers into template functions.
func funcMap() template.FuncMap {
	return template.FuncMap{
		"versionClass":     versionClass,
		"elapsed":          formatElapsed,
		"stepNodes":        stepNodes,
		"barWidth":         barWidth,
		"failingByVersion": failingByVersion,
		"dict":             dict,
	}
}

// dict builds a map from alternating key/value args, letting a template pass
// multiple values to a sub-template (here: a version card and its failing count).
func dict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments (%d)", len(pairs))
	}
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key %d is not a string", i)
		}
		m[key] = pairs[i+1]
	}
	return m, nil
}

// versionClass maps a friendly version to its badge color class (b-v1/b-v2/b-v3),
// defaulting to b-v1 like app.js.
func versionClass(version string) string {
	switch version {
	case "v2":
		return "b-v2"
	case "v3":
		return "b-v3"
	default:
		return "b-v1"
	}
}

// formatElapsed renders seconds as m:ss, matching app.js formatElapsed.
func formatElapsed(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

// stepNodes computes the per-node stepper state for an order, matching the
// logic of app.js renderStepper (done/current/error + connector fill).
func stepNodes(o Order) []stepNode {
	nodes := make([]stepNode, len(o.Steps))
	for i, label := range o.Steps {
		n := stepNode{Label: string(label)}
		switch {
		case o.Done || i < o.CurrentStep:
			n.Class, n.Glyph = "done", "✓"
		case i == o.CurrentStep:
			if o.Failing {
				n.Class, n.Glyph = "err", "✕"
			} else {
				n.Class = "cur"
			}
		}
		if i < len(o.Steps)-1 {
			n.HasConn = true
			n.ConnFill = o.Done || i < o.CurrentStep
		}
		nodes[i] = n
	}
	return nodes
}

// barWidth clamps the traffic percentage to [0,100] for the version bar width.
func barWidth(pct int) int {
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// failingByVersion counts failing orders per friendly version, so a version
// card can flag its own slice (mirrors app.js renderVersions).
func failingByVersion(state DashboardState) map[string]int {
	counts := make(map[string]int)
	for _, o := range state.Orders {
		if o.Failing {
			counts[o.Version]++
		}
	}
	return counts
}
