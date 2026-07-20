package main

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
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
		"width: 100%",
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
		// Design-system face + layout tokens (sty_f5c3b40f).
		`font-weight: 300 700`,
		`font-display: swap`,
		`format("woff2")`,
		`/assets/fonts/space-grotesk-latin.woff2`,
		`U+0000-00FF`,
		`font: var(--text-base)/1.5 var(--display)`,
		`--display: "Space Grotesk", "Trebuchet MS", "Segoe UI", sans-serif`,
		`--font-body: var(--display)`,
		`font-size: var(--text-app-h1)`,
		`letter-spacing: var(--tracking-display)`,
		`font-size: var(--text-meta)`,
		`letter-spacing: var(--tracking-heading)`,
		`font-size: var(--text-th)`,
		`letter-spacing: var(--tracking-th)`,
		`font-size: var(--text-badge)`,
		`letter-spacing: var(--tracking-badge)`,
		`font-size: var(--text-chip)`,
		`var(--gutter-mobile)`,
		`--expand-bg: #fafbfc`,
		`--expand-bg: #121418`,
		`--gutter-mobile: 1rem`,
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
	// Literals that must live only in :root token definitions.
	for _, lit := range []string{"12.5px", "0.06em", "0.04em"} {
		if n := strings.Count(s, lit); n != 1 {
			t.Errorf("%q should appear exactly once (token definition only), got %d", lit, n)
		}
	}
	for _, bad := range []string{
		"width: 90%",
		"-apple-system",
		"BlinkMacSystemFont",
		"fonts.googleapis",
		"fonts.gstatic",
		"//cdn",
		"details.stack[open]",
		"☾/☀",
	} {
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

func TestFontAsset(t *testing.T) {
	mux := http.NewServeMux()
	registerAssets(mux)

	req := httptest.NewRequest(http.MethodGet, fontURLPath, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "font/woff2" {
		t.Errorf("Content-Type = %q, want font/woff2", ct)
	}
	cc := rr.Header().Get("Cache-Control")
	if !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable", cc)
	}
	body := rr.Body.Bytes()
	if len(body) == 0 {
		t.Fatal("empty font body")
	}
	if !bytes.HasPrefix(body, []byte("wOF2")) {
		t.Errorf("body magic = %q, want wOF2", body[:min(4, len(body))])
	}

	// HEAD should also succeed with the same headers and no body requirement.
	reqH := httptest.NewRequest(http.MethodHead, fontURLPath, nil)
	rrH := httptest.NewRecorder()
	mux.ServeHTTP(rrH, reqH)
	if rrH.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", rrH.Code)
	}
	if rrH.Header().Get("Content-Type") != "font/woff2" {
		t.Errorf("HEAD Content-Type = %q", rrH.Header().Get("Content-Type"))
	}
}
