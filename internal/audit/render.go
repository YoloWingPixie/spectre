package audit

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/YoloWingPixie/spectre/internal/webui"
)

const imageEmbedLimit = 3 * 1024 * 1024

var imageMIME = map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml"}
var anchorPattern = regexp.MustCompile(`[^a-z0-9]+`)
var codePattern = regexp.MustCompile("`([^`]+)`")
var inlineReference = regexp.MustCompile(`^[\w.@+()[\]-]+(?:/[\w.@+()[\]-]+)+\.\w+:\d+(?:-\d+)?$`)
var severityOrder = [...]severity{critical, high, medium, low}

func anchor(id string) string {
	return "f-" + strings.Trim(anchorPattern.ReplaceAllString(strings.ToLower(id), "-"), "-")
}
func fieldID(id findingID, part string) string { return anchor(string(id)) + "-" + part }
func severityRank(s severity) int              { return slices.Index(severityOrder[:], s) }
func label(value any) string {
	labels := map[string]string{"critical": "Critical", "high": "High", "medium": "Medium", "low": "Low", "open": "Open", "fixed": "Fixed", "accepted": "Accepted", "wont-fix": "Won't fix", "S": "Small", "M": "Medium", "L": "Large", "med": "Medium"}
	return labels[fmt.Sprint(value)]
}
func reference(root, distro string, ref location) template.HTML {
	line, col := ref.Line, ref.Col
	if line == 0 {
		line = 1
	}
	if col == 0 {
		col = 1
	}
	text := ref.Path
	if ref.Line != 0 {
		text += fmt.Sprintf(":%d", ref.Line)
	}
	if ref.EndLine != 0 && ref.EndLine != ref.Line {
		text += fmt.Sprintf("-%d", ref.EndLine)
	}
	path := strings.TrimRight(root, "/") + "/" + strings.TrimPrefix(ref.Path, "./")
	path = strings.NewReplacer("%", "%25", " ", "%20", "#", "%23", "?", "%3F").Replace(path)
	href := fmt.Sprintf("vscode://vscode-remote/wsl+%s%s:%d:%d", strings.ReplaceAll(url.QueryEscape(distro), "+", "%20"), path, line, col)
	copy := fmt.Sprintf("%s:%d", ref.Path, line)
	return template.HTML(fmt.Sprintf(`<span class="ref"><a class="fref" href="%s" data-path="%s" data-line="%d" data-col="%d" title="Open %s in your editor">%s</a><button type="button" class="copy" data-copy="%s" aria-label="Copy %s" title="Copy path:line">⧉</button></span>`, html.EscapeString(href), html.EscapeString(ref.Path), line, col, html.EscapeString(text), html.EscapeString(text), html.EscapeString(copy), html.EscapeString(copy)))
}
func markdown(text, root, distro string) template.HTML {
	var out strings.Builder
	previous := 0
	for _, match := range codePattern.FindAllStringSubmatchIndex(text, -1) {
		out.WriteString(html.EscapeString(text[previous:match[0]]))
		code := text[match[2]:match[3]]
		previous = match[1]
		if inlineReference.MatchString(code) {
			ref, ok := parseTouch(code)
			if ok && relativeFile(ref.Path) {
				out.WriteString(string(reference(root, distro, ref)))
				continue
			}
		}
		out.WriteString("<code>" + html.EscapeString(code) + "</code>")
	}
	out.WriteString(html.EscapeString(text[previous:]))
	return template.HTML(out.String())
}
func searchText(f finding) string {
	parts := []string{string(f.ID), f.Title, f.Summary, f.StatusNote, label(f.Severity), label(f.Status), f.Location.Path}
	parts = append(parts, f.Tags...)
	for _, e := range f.Evidence {
		parts = append(parts, e.Path, e.Note, e.Excerpt, e.Caption, e.Command, e.Output)
	}
	parts = append(parts, f.Current.Description)
	parts = append(parts, f.Current.Pros...)
	parts = append(parts, f.Current.Cons...)
	for _, c := range f.COAs {
		parts = append(parts, c.Name, c.Description)
		parts = append(parts, c.Pros...)
		parts = append(parts, c.Cons...)
		parts = append(parts, c.Touches...)
	}
	return strings.ToLower(strings.Join(parts, " "))
}
func collectImages(r report, baseDir, outFile string, contained bool) (map[string]template.URL, error) {
	paths := []string{}
	total := int64(0)
	for _, f := range r.Findings {
		for _, e := range f.Evidence {
			if e.Type == screenshotEvidence && !slices.Contains(paths, e.Path) {
				paths = append(paths, e.Path)
			}
		}
	}
	for _, path := range paths {
		resolved := screenshotPath(baseDir, path)
		if contained {
			real, err := filepath.EvalSymlinks(resolved)
			if err != nil {
				return nil, err
			}
			root, err := filepath.EvalSymlinks(baseDir)
			if err != nil {
				return nil, err
			}
			rel, err := filepath.Rel(root, real)
			if err != nil || !filepath.IsLocal(rel) {
				return nil, invalid("Viewer screenshots must stay inside the audit directory")
			}
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, invalid("Screenshot must be a regular file")
		}
		total += info.Size()
	}
	images := map[string]template.URL{}
	for i, path := range paths {
		resolved := screenshotPath(baseDir, path)
		data, err := os.ReadFile(resolved)
		if err != nil {
			return nil, err
		}
		mime, ok := imageMIME[strings.ToLower(filepath.Ext(path))]
		if !ok {
			return nil, invalid("Unsupported screenshot format")
		}
		if total <= imageEmbedLimit || outFile == "" {
			images[path] = template.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data))
			continue
		}
		directory := strings.TrimSuffix(filepath.Base(outFile), filepath.Ext(outFile)) + ".assets"
		name := fmt.Sprintf("%02d-%s", i+1, filepath.Base(path))
		if err := os.MkdirAll(filepath.Join(filepath.Dir(outFile), directory), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(outFile), directory, name), data, 0o644); err != nil {
			return nil, err
		}
		images[path] = template.URL(url.PathEscape(directory) + "/" + url.PathEscape(name))
	}
	return images, nil
}

func screenshotPath(baseDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, filepath.FromSlash(path))
}

type countView struct {
	Value string
	Label string
	Total int
	Open  int
}
type areaView struct {
	area
	Findings []finding
	Counts   []int
	Open     int
	Closed   int
}
type priorityView struct {
	Finding finding
	Why     string
}
type viewerConnection struct {
	Token          string
	ReportRevision string
}
type renderData struct {
	report
	CSS        template.CSS
	JS         template.JS
	ThemeJS    template.JS
	Header     webui.Header
	Distro     string
	Viewer     *viewerConnection
	Groups     []areaView
	Severities []countView
	Statuses   []countView
	Open       int
	Closed     int
	Priorities []priorityView
}

func renderReport(r report, images map[string]template.URL, viewer *viewerConnection) ([]byte, error) {
	distro := r.WSLDistro
	if distro == "" {
		distro = os.Getenv("WSL_DISTRO_NAME")
	}
	if distro == "" {
		distro = "Ubuntu"
	}
	css, err := resources.ReadFile("assets/style.css")
	if err != nil {
		return nil, err
	}
	var js strings.Builder
	for _, path := range []string{"assets/links-helpers.js", "assets/feedback-helpers.js", "assets/script.js", "assets/feedback.js"} {
		data, err := resources.ReadFile(path)
		if err != nil {
			return nil, err
		}
		js.Write(data)
		js.WriteByte('\n')
	}
	home := ""
	if viewer != nil {
		home = "/"
	}
	data := renderData{report: r, CSS: webui.Styles(css), JS: template.JS(js.String()), Distro: distro, Viewer: viewer, ThemeJS: webui.ThemeScript(), Header: webui.Header{Mode: webui.Audit, Name: r.Project, Home: home}}
	for _, value := range severityOrder {
		c := countView{Value: string(value), Label: label(value)}
		for _, f := range r.Findings {
			if f.Severity == value {
				c.Total++
				if f.Status == open {
					c.Open++
				}
			}
		}
		data.Severities = append(data.Severities, c)
		data.Open += c.Open
	}
	data.Closed = len(r.Findings) - data.Open
	for _, value := range []findingStatus{open, fixed, accepted, wontFix} {
		c := countView{Value: string(value), Label: label(value)}
		for _, f := range r.Findings {
			if f.Status == value {
				c.Total++
			}
		}
		data.Statuses = append(data.Statuses, c)
	}
	for _, a := range r.Areas {
		group := areaView{area: a, Counts: make([]int, len(severityOrder))}
		for _, f := range r.Findings {
			if f.Area == a.Key {
				group.Findings = append(group.Findings, f)
				if f.Status == open {
					group.Open++
					group.Counts[severityRank(f.Severity)]++
				} else {
					group.Closed++
				}
			}
		}
		slices.SortStableFunc(group.Findings, func(a, b finding) int {
			if (a.Status == open) != (b.Status == open) {
				if a.Status == open {
					return -1
				}
				return 1
			}
			return severityRank(a.Severity) - severityRank(b.Severity)
		})
		data.Groups = append(data.Groups, group)
	}
	for _, p := range r.FixFirst {
		for _, f := range r.Findings {
			if f.ID == p.ID {
				why := p.Why
				if why == "" {
					why = f.Summary
				}
				data.Priorities = append(data.Priorities, priorityView{f, why})
			}
		}
	}
	funcs := template.FuncMap{
		"md": func(text string) template.HTML { return markdown(text, r.RepoRoot, distro) }, "label": label,
		"anchor": func(id findingID) string { return anchor(string(id)) }, "fid": fieldID, "search": searchText,
		"ref": func(ref location) template.HTML { return reference(r.RepoRoot, distro, ref) },
		"evidenceRef": func(e evidence) template.HTML {
			return reference(r.RepoRoot, distro, location{e.Path, e.Line, e.EndLine, e.Col})
		},
		"touch": func(text string) template.HTML { ref, _ := parseTouch(text); return reference(r.RepoRoot, distro, ref) },
		"image": func(path string) template.URL { return images[path] }, "letter": func(i int) string { return string(rune('A' + i)) },
		"inc": func(i int) int { return i + 1 }, "plain": func(s string) string { return strings.ReplaceAll(s, "`", "") },
	}
	t, err := webui.Templates(template.New("report").Funcs(funcs))
	if err == nil {
		t, err = t.ParseFS(resources, "assets/report.html")
	}
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := t.ExecuteTemplate(&out, "report.html", data); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
