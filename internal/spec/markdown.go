package spec

import (
	"html"
	"regexp"
	"strings"
)

// A small Markdown renderer for the spec files: ATX headings, paragraphs,
// nested bullet and numbered lists, pipe tables, fenced code, block quotes,
// rules, and inline code, bold, italic and links. Raw HTML is always escaped.
// Plain text runs through the Linker so every spec ID becomes a link.

var (
	reFence    = regexp.MustCompile("^\\s*```")
	reHeading  = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	reRule     = regexp.MustCompile(`^\s{0,3}([-*_])(\s*[-*_]){2,}\s*$`)
	reListItem = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(.*)$`)
	reTableSep = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)
)

// renderer holds per-render state: the linker keeps the "last prefix" used to
// expand short requirement IDs while it walks the text in order.
type renderer struct {
	lk *linkState
	sb strings.Builder
}

// RenderMarkdown renders md to HTML, linking IDs through lk (nil: no links).
func RenderMarkdown(md string, lk *Linker) string {
	r := &renderer{lk: lk.state()}
	r.blocks(splitLines(md))
	return r.sb.String()
}

// RenderInline renders a single line of Markdown without a paragraph wrapper.
func RenderInline(md string, lk *Linker) string {
	r := &renderer{lk: lk.state()}
	r.inline(md)
	return r.sb.String()
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\t", "    ")
	return strings.Split(s, "\n")
}

func blank(l string) bool { return strings.TrimSpace(l) == "" }

func indentOf(l string) int { return len(l) - len(strings.TrimLeft(l, " ")) }

func (r *renderer) w(s string) { r.sb.WriteString(s) }

// blocks renders a sequence of block-level lines.
func (r *renderer) blocks(lines []string) {
	for i := 0; i < len(lines); {
		l := lines[i]
		switch {
		case blank(l):
			i++
		case reFence.MatchString(l):
			i = r.fence(lines, i)
		case reHeading.MatchString(l):
			m := reHeading.FindStringSubmatch(l)
			lvl := string(rune('0' + len(m[1])))
			r.w("<h" + lvl + ` id="` + html.EscapeString(Slug(m[2])) + `">`)
			r.inline(m[2])
			r.w("</h" + lvl + ">\n")
			i++
		case reRule.MatchString(l) && !reListItem.MatchString(l):
			r.w("<hr>\n")
			i++
		case isTableStart(lines, i):
			i = r.table(lines, i)
		case strings.HasPrefix(strings.TrimLeft(l, " "), ">"):
			j := i
			var q []string
			for j < len(lines) && strings.HasPrefix(strings.TrimLeft(lines[j], " "), ">") {
				t := strings.TrimPrefix(strings.TrimLeft(lines[j], " "), ">")
				q = append(q, strings.TrimPrefix(t, " "))
				j++
			}
			r.w("<blockquote>")
			r.blocks(q)
			r.w("</blockquote>\n")
			i = j
		case reListItem.MatchString(l):
			i = r.list(lines, i)
		default:
			j := i
			var p []string
			for j < len(lines) && !blank(lines[j]) && (j == i || !startsBlock(lines, j)) {
				p = append(p, strings.TrimSpace(lines[j]))
				j++
			}
			r.w("<p>")
			r.inline(strings.Join(p, "\n"))
			r.w("</p>\n")
			i = j
		}
	}
}

func startsBlock(lines []string, i int) bool {
	l := lines[i]
	return reFence.MatchString(l) || reHeading.MatchString(l) || reListItem.MatchString(l) ||
		isTableStart(lines, i) || strings.HasPrefix(strings.TrimLeft(l, " "), ">") || reRule.MatchString(l)
}

func (r *renderer) fence(lines []string, i int) int {
	ind := indentOf(lines[i])
	j := i + 1
	var code []string
	for j < len(lines) && !reFence.MatchString(lines[j]) {
		l := lines[j]
		if indentOf(l) >= ind {
			l = l[ind:]
		} else {
			l = strings.TrimLeft(l, " ")
		}
		code = append(code, l)
		j++
	}
	r.w("<pre><code>")
	r.w(html.EscapeString(strings.Join(code, "\n")))
	r.w("</code></pre>\n")
	if j < len(lines) {
		j++ // closing fence
	}
	return j
}

func isTableStart(lines []string, i int) bool {
	if i+1 >= len(lines) {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(lines[i]), "|") && reTableSep.MatchString(lines[i+1]) &&
		strings.Contains(lines[i+1], "-")
}

// SplitCells splits a pipe-table row into trimmed cells. Pipes inside code
// spans or escaped as \| do not split.
func SplitCells(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(row, "|")
	if strings.HasSuffix(row, "|") && !strings.HasSuffix(row, `\|`) {
		row = row[:len(row)-1]
	}
	var cells []string
	var cur strings.Builder
	inCode := false
	for i := 0; i < len(row); i++ {
		c := row[i]
		switch {
		case c == '\\' && i+1 < len(row) && row[i+1] == '|':
			cur.WriteByte('|')
			i++
		case c == '`':
			inCode = !inCode
			cur.WriteByte(c)
		case c == '|' && !inCode:
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

func (r *renderer) table(lines []string, i int) int {
	head := SplitCells(lines[i])
	r.w(`<div class="tw"><table><thead><tr>`)
	for _, c := range head {
		r.w("<th>")
		r.inline(c)
		r.w("</th>")
	}
	r.w("</tr></thead><tbody>\n")
	j := i + 2
	for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|") {
		cells := SplitCells(lines[j])
		idAttr := ""
		if len(cells) > 0 && reDEID.MatchString(cells[0]) {
			idAttr = ` id="` + cells[0] + `"`
		}
		r.w("<tr" + idAttr + ">")
		for _, c := range cells {
			r.w("<td>")
			r.inline(c)
			r.w("</td>")
		}
		r.w("</tr>\n")
		j++
	}
	r.w("</tbody></table></div>\n")
	return j
}

// list renders a (possibly nested) list starting at line i.
func (r *renderer) list(lines []string, i int) int {
	m := reListItem.FindStringSubmatch(lines[i])
	base := len(m[1])
	ordered := m[2][0] >= '0' && m[2][0] <= '9'
	tag := "ul"
	if ordered {
		tag = "ol"
	}
	r.w("<" + tag + ">\n")
	j := i
	for j < len(lines) {
		m := reListItem.FindStringSubmatch(lines[j])
		if m == nil || len(m[1]) != base || (m[2][0] >= '0' && m[2][0] <= '9') != ordered {
			break
		}
		content := []string{m[3]}
		cont := len(m[1]) + len(m[2]) + 1
		k := j + 1
		loose := false
		for k < len(lines) {
			l := lines[k]
			if blank(l) {
				// A blank line continues the item when the next non-blank
				// line is indented past the marker.
				n := k + 1
				for n < len(lines) && blank(lines[n]) {
					n++
				}
				if n < len(lines) && indentOf(lines[n]) > base {
					for ; k < n; k++ {
						content = append(content, "")
					}
					loose = true
					continue
				}
				break
			}
			ind := indentOf(l)
			if ind <= base && (reListItem.MatchString(l) || startsBlock(lines, k)) {
				break
			}
			if ind <= base && !reListItem.MatchString(l) {
				// lazy paragraph continuation
				content = append(content, strings.TrimSpace(l))
				k++
				continue
			}
			strip := cont
			if ind < strip {
				strip = ind
			}
			content = append(content, l[strip:])
			k++
		}
		r.w("<li>")
		r.item(content, loose)
		r.w("</li>\n")
		j = k
		// Skip blank lines between items of the same list.
		n := j
		for n < len(lines) && blank(lines[n]) {
			n++
		}
		if n < len(lines) && n > j {
			if mm := reListItem.FindStringSubmatch(lines[n]); mm != nil && len(mm[1]) == base {
				j = n
			}
		}
	}
	r.w("</" + tag + ">\n")
	return j
}

// item renders one list item's content; a tight single paragraph has no <p>.
func (r *renderer) item(content []string, loose bool) {
	if !loose {
		// Leading paragraph lines are inline; the rest (nested lists) are blocks.
		n := 0
		for n < len(content) && !blank(content[n]) && (n == 0 || !startsBlock(content, n)) {
			n++
		}
		var p []string
		for _, l := range content[:n] {
			p = append(p, strings.TrimSpace(l))
		}
		r.inline(strings.Join(p, "\n"))
		if n < len(content) {
			r.w("\n")
			r.blocks(content[n:])
		}
		return
	}
	r.blocks(content)
}

// ---------------------------------------------------------------- inline

func (r *renderer) inline(s string) {
	var text strings.Builder
	flush := func() {
		if text.Len() > 0 {
			r.w(r.lk.link(text.String()))
			text.Reset()
		}
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && strings.ContainsRune("\\`*_[]()#|<>-.!", rune(s[i+1])):
			text.WriteByte(s[i+1])
			i += 2
			continue
		case c == '`':
			n := 1
			for i+n < len(s) && s[i+n] == '`' {
				n++
			}
			fence := s[i : i+n]
			if end := strings.Index(s[i+n:], fence); end >= 0 {
				flush()
				code := strings.TrimSpace(s[i+n : i+n+end])
				r.w("<code>" + r.lk.link(code) + "</code>")
				i += n + end + n
				continue
			}
		case c == '*' || c == '_':
			if c == '_' && i > 0 && isWord(s[i-1]) {
				break
			}
			if strings.HasPrefix(s[i:], string([]byte{c, c})) {
				if end := strings.Index(s[i+2:], string([]byte{c, c})); end > 0 {
					flush()
					r.w("<strong>")
					r.inline(s[i+2 : i+2+end])
					r.w("</strong>")
					i += 2 + end + 2
					continue
				}
			} else if i+1 < len(s) && s[i+1] != ' ' {
				if end := closingEmph(s[i+1:], c); end > 0 {
					flush()
					r.w("<em>")
					r.inline(s[i+1 : i+1+end])
					r.w("</em>")
					i += 1 + end + 1
					continue
				}
			}
		case c == '[':
			if tEnd := strings.Index(s[i:], "]("); tEnd > 0 {
				if uEnd := strings.IndexByte(s[i+tEnd+2:], ')'); uEnd >= 0 {
					label := s[i+1 : i+tEnd]
					url := s[i+tEnd+2 : i+tEnd+2+uEnd]
					flush()
					r.w(`<a href="` + html.EscapeString(mapHref(url)) + `">`)
					inner := &renderer{lk: &linkState{}}
					inner.inline(label)
					r.w(inner.sb.String() + "</a>")
					i += tEnd + 2 + uEnd + 1
					continue
				}
			}
		case c == '\n':
			text.WriteByte(' ')
			i++
			continue
		}
		text.WriteByte(c)
		i++
	}
	flush()
}

func isWord(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// closingEmph finds a closing single * or _ that is not doubled and not
// preceded by a space.
func closingEmph(s string, c byte) int {
	for k := 1; k < len(s); k++ {
		if s[k] == '`' {
			if e := strings.IndexByte(s[k+1:], '`'); e >= 0 {
				k += e + 1
				continue
			}
		}
		if s[k] != c || s[k-1] == ' ' {
			continue
		}
		if k+1 < len(s) && s[k+1] == c {
			k++
			continue
		}
		if c == '_' && k+1 < len(s) && isWord(s[k+1]) {
			continue
		}
		return k
	}
	return -1
}

// mapHref turns links to the spec's own .md files into viewer routes.
func mapHref(u string) string {
	base, frag, _ := strings.Cut(u, "#")
	base = strings.TrimPrefix(base, "./")
	switch base {
	case "README.md":
		return "/readme"
	case "requirements.md":
		return "/req"
	case "design.md":
		return "/design/1"
	case "scenarios.md":
		return "/scenarios/1"
	case "tasks.md":
		return "/tasks/1"
	case "open-decisions.md":
		return "/od"
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "#") || strings.HasPrefix(u, "/") {
		return u
	}
	if base == "" && frag != "" {
		return "#" + frag
	}
	return "#"
}

// Slug makes an HTML id from a heading.
func Slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
