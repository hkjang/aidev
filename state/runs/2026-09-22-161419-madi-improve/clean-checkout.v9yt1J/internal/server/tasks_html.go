package server

import (
	"bytes"
	"sort"
	"strings"

	xhtml "golang.org/x/net/html"
)

// Only explicit editor task markup is indexed. Ordinary HTML checkboxes and
// literal examples in code/script/template blocks are never promoted to tasks.
// Offsets refer to the original Markdown bytes, so several tasks on one HTML
// line remain individually addressable without rewriting the surrounding table.
type htmlTaskRange struct {
	start, openingEnd, end, insert int
	opening                        xhtml.Token
	refs                           [][2]int
	ids                            []string
	label                          strings.Builder
	closed                         bool
}
type htmlTaskFrame struct {
	tag        string
	skip, list bool
	task       *htmlTaskRange
	ref        *htmlTaskRange
	start      int
}

func indexHTMLTasks(source []byte, start, end int) []markdownTask {
	if start < 0 || end > len(source) || start >= end {
		return nil
	}
	z := xhtml.NewTokenizer(bytes.NewReader(source[start:end]))
	stack := []htmlTaskFrame{}
	all := []*htmlTaskRange{}
	offset := start
	currentTask := func() *htmlTaskRange {
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].task != nil {
				return stack[i].task
			}
		}
		return nil
	}
	for {
		kind := z.Next()
		if kind == xhtml.ErrorToken {
			break
		}
		rawLen := len(z.Raw())
		at := offset
		offset += rawLen
		if len(stack) > 256 || len(all) > 10000 {
			return nil
		}
		skip, inRef := false, false
		for _, f := range stack {
			skip = skip || f.skip
			inRef = inRef || f.ref != nil
		}
		token := z.Token()
		switch kind {
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			attrs := map[string]string{}
			duplicate := false
			for _, a := range token.Attr {
				if _, ok := attrs[a.Key]; ok {
					duplicate = true
				}
				attrs[a.Key] = a.Val
			}
			frame := htmlTaskFrame{tag: token.Data, start: at, skip: skip || oneOf(token.Data, "code", "pre", "script", "style", "textarea", "template")}
			if !frame.skip && !duplicate {
				frame.list = token.Data == "ul" && attrs["data-type"] == "taskList"
				if token.Data == "li" && attrs["data-type"] == "taskItem" && len(stack) > 0 && stack[len(stack)-1].list && oneOf(attrs["data-checked"], "true", "false") {
					frame.task = &htmlTaskRange{start: at, openingEnd: offset, opening: token, insert: offset}
					all = append(all, frame.task)
				}
				if task := currentTask(); task != nil && !inRef {
					if token.Data == "a" {
						if id, ok := strings.CutPrefix(attrs["href"], "/app/tasks?task="); ok && validID(id) {
							frame.ref = task
							task.ids = append(task.ids, strings.ToLower(id))
						}
					}
					if oneOf(token.Data, "p", "br", "div", "ul", "ol") {
						task.label.WriteByte(' ')
					}
				}
			}
			if kind != xhtml.SelfClosingTagToken && !oneOf(token.Data, "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr") {
				stack = append(stack, frame)
			}
		case xhtml.TextToken:
			if task := currentTask(); task != nil && !skip && !inRef {
				task.label.WriteString(token.Data)
			}
		case xhtml.EndTagToken:
			// Fail closed for malformed task containers rather than infer browser
			// error-recovery tree positions when mutating the original source.
			if len(stack) == 0 || stack[len(stack)-1].tag != token.Data {
				continue
			}
			frame := stack[len(stack)-1]
			if frame.ref != nil {
				frame.ref.refs = append(frame.ref.refs, [2]int{frame.start, offset})
			}
			if task := currentTask(); task != nil && !skip && token.Data == "p" && task.insert == task.openingEnd {
				task.insert = at
			}
			if frame.task != nil {
				frame.task.end = offset
				frame.task.closed = true
			}
			stack = stack[:len(stack)-1]
		}
	}
	out := []markdownTask{}
	for _, v := range all {
		if !v.closed {
			continue
		}
		done := false
		for _, a := range v.opening.Attr {
			if a.Key == "data-checked" {
				done = a.Val == "true"
			}
		}
		t := markdownTask{Text: strings.Join(strings.Fields(v.label.String()), " "), Done: done, Start: v.start, Line: bytes.Count(source[:v.start], []byte{'\n'}), EndLine: bytes.Count(source[:v.end], []byte{'\n'}), HTML: v}
		if len(v.ids) > 0 {
			t.ID = v.ids[0]
			t.Ambiguous = len(v.ids) > 1
		}
		out = append(out, t)
	}
	return out
}

func updateHTMLTask(markdown string, target markdownTask, id string, done, separate bool) string {
	v := target.HTML
	token := v.opening
	for i := range token.Attr {
		if token.Attr[i].Key == "data-checked" {
			token.Attr[i].Val = map[bool]string{true: "true", false: "false"}[done]
		}
	}
	type edit struct {
		start, end int
		text       string
	}
	edits := []edit{{v.start, v.openingEnd, token.String()}}
	if target.ID == "" || separate {
		if separate {
			for _, ref := range v.refs {
				edits = append(edits, edit{ref[0], ref[1], ""})
			}
		}
		edits = append(edits, edit{v.insert, v.insert, ` <a href="/app/tasks?task=` + id + `">작업</a>`})
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	for _, e := range edits {
		markdown = markdown[:e.start] + e.text + markdown[e.end:]
	}
	return markdown
}
