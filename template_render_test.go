package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"time"
)

func TestIndexTemplateRender(t *testing.T) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		t.Fatal(err)
	}
	data := pageData{
		Generated:   time.Now(),
		ShowStopped: true,
		Stacks: []stackView{{
			Name: "demo",
			Containers: []containerView{
				{ID: "abc", Name: "up", State: "running", Running: true},
				{ID: "def", Name: "down", State: "exited", Running: false},
			},
		}},
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "index.html", data); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{
		"--content-max: 1600px",
		"width: 90%",
		"aria-label",
		"hidden-action",
		"data-stack-actions",
		"table-layout: fixed",
		".trunc",
		"text-overflow: ellipsis",
		// Shared column widths (defined once for all stack tables).
		"th:nth-child(1), td:nth-child(1)",
		"th:nth-child(7), td:nth-child(7)",
		// Narrow media hides service = column 2 (not state = 3).
		"th:nth-child(2), td:nth-child(2) { display: none; }",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Count(s, "table-layout: fixed") != 1 {
		t.Errorf("table-layout:fixed should appear once, got %d", strings.Count(s, "table-layout: fixed"))
	}
	// Name/service cells carry title for full value on hover.
	if !strings.Contains(s, `class="trunc" title="up"`) {
		t.Error("name cell should be .trunc with title")
	}
	for _, bad := range []string{"Space Grotesk", "details.stack[open]", "☾/☀"} {
		if strings.Contains(s, bad) {
			t.Errorf("unexpected %q", bad)
		}
	}
	// Running container: start hidden; stop visible.
	if !strings.Contains(s, `class="small hidden-action" data-ctr-action="start" data-id="abc"`) {
		t.Error("running container should hide start")
	}
	if !strings.Contains(s, `class="small" data-ctr-action="stop" data-id="abc"`) {
		t.Error("running container should show stop")
	}
	// Stopped container: start visible; stop hidden.
	if !strings.Contains(s, `class="small" data-ctr-action="start" data-id="def"`) {
		t.Error("stopped container should show start")
	}
	if !strings.Contains(s, `class="small hidden-action" data-ctr-action="stop" data-id="def"`) {
		t.Error("stopped container should hide stop")
	}
	// Mixed stack: both start and stop available (not both hidden).
	if !strings.Contains(s, `class="small" data-stack-action="start" data-stack="demo"`) {
		t.Error("mixed stack should show start (any stopped)")
	}
	if !strings.Contains(s, `class="small" data-stack-action="stop" data-stack="demo"`) {
		t.Error("mixed stack should show stop (any running)")
	}
}
