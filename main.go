package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

//go:embed templates/*
var templateFS embed.FS

const (
	labelComposeProject = "com.docker.compose.project"
	labelComposeService = "com.docker.compose.service"
	noProjectLabel      = "(no project)"
	listenAddr          = ":8888"
)

type portView struct {
	HostIP   string
	HostPort string
	ContPort string
	Proto    string
}

type containerView struct {
	ID      string
	Name    string
	Service string
	State   string
	Status  string
	Ports   []portView
}

type stackView struct {
	Name       string
	Containers []containerView
}

type pageData struct {
	Stacks    []stackView
	Generated time.Time
	Error     string
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("docker client: %v", err)
	}
	defer cli.Close()

	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := pageData{Generated: time.Now()}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		list, err := cli.ContainerList(ctx, container.ListOptions{All: false})
		if err != nil {
			data.Error = err.Error()
			log.Printf("container list: %v", err)
		} else {
			data.Stacks = groupByProject(list)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("render: %v", err)
		}
	})

	addr := listenAddr
	if v := os.Getenv("PORTSIDE_ADDR"); v != "" {
		addr = v
	}
	log.Printf("portside listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func groupByProject(list []types.Container) []stackView {
	groups := map[string][]containerView{}
	for _, c := range list {
		project := c.Labels[labelComposeProject]
		if project == "" {
			project = noProjectLabel
		}
		groups[project] = append(groups[project], toContainerView(c))
	}

	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]stackView, 0, len(names))
	for _, name := range names {
		rows := groups[name]
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].Name < rows[j].Name
		})
		out = append(out, stackView{Name: name, Containers: rows})
	}
	return out
}

func toContainerView(c types.Container) containerView {
	name := c.ID
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}
	ports := make([]portView, 0, len(c.Ports))
	for _, p := range c.Ports {
		if p.PublicPort == 0 {
			continue
		}
		ports = append(ports, portView{
			HostIP:   p.IP,
			HostPort: itoa(p.PublicPort),
			ContPort: itoa(p.PrivatePort),
			Proto:    p.Type,
		})
	}
	return containerView{
		ID:      c.ID,
		Name:    name,
		Service: c.Labels[labelComposeService],
		State:   c.State,
		Status:  c.Status,
		Ports:   ports,
	}
}

func itoa(u uint16) string {
	if u == 0 {
		return "0"
	}
	var b [6]byte
	i := len(b)
	for u > 0 {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
	}
	return string(b[i:])
}
