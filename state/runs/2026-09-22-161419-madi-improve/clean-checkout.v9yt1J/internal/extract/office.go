package extract

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Office reads OOXML only. It never evaluates formulae, fetches relationships,
// loads external entities, opens embedded objects, or executes VBA. Legacy
// binary DOC/XLS/PPT and encrypted containers are explicitly unsupported.
func Office(ctx context.Context, source io.ReaderAt, size int64, format string, limits Limits) (Result, error) {
	c := newCollector(limits)
	if size < 1 || size > limits.InputBytes {
		return c.result, ErrLimit
	}
	z, err := zip.NewReader(source, size)
	if err != nil {
		return c.result, errors.New("암호화되지 않은 올바른 OOXML 파일이 필요합니다")
	}
	files := map[string]*zip.File{}
	var total uint64
	if len(z.File) > 10000 {
		return c.result, ErrLimit
	}
	for _, f := range z.File {
		if f.Name == "" || strings.Contains(f.Name, "\\") || strings.HasPrefix(f.Name, "/") || strings.HasPrefix(f.Name, "../") || f.Name == ".." || path.Clean(strings.TrimSuffix(f.Name, "/")) != strings.TrimSuffix(f.Name, "/") || f.Mode()&0111 != 0 && !f.FileInfo().IsDir() || f.Mode()&os.ModeType != 0 && !f.FileInfo().IsDir() {
			return c.result, errors.New("Office 압축 경로·파일 형식이 안전하지 않습니다")
		}
		if _, exists := files[f.Name]; exists {
			return c.result, errors.New("Office 압축 파일에 중복 경로가 있습니다")
		}
		files[f.Name] = f
		if f.UncompressedSize64 > uint64(limits.ExpandedBytes) || total > uint64(limits.ExpandedBytes)-f.UncompressedSize64 {
			return c.result, ErrLimit
		}
		total += f.UncompressedSize64
		if strings.HasSuffix(strings.ToLower(f.Name), "vbaproject.bin") || strings.Contains(f.Name, "/embeddings/") {
			c.result.Warnings = appendUnique(c.result.Warnings, "매크로·내장 개체는 실행하거나 추출하지 않았습니다")
		}
	}
	if files["[Content_Types].xml"] == nil {
		return c.result, errors.New("Office 형식 선언이 없습니다")
	}
	switch format {
	case "docx":
		err = officeWord(ctx, files, c)
	case "pptx":
		err = officePresentation(ctx, files, c)
	case "xlsx":
		err = officeWorkbook(ctx, files, c)
	default:
		err = errors.New("DOCX·PPTX·XLSX 파일을 선택하세요. 기존 DOC·PPT·XLS와 암호화 파일은 지원하지 않습니다")
	}
	return c.result, err
}

func appendUnique(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

// Decoder does not resolve DTD entities. Reject directives outright so a file
// with entity declarations is never silently interpreted as ordinary content.
func walkXML(ctx context.Context, file *zip.File, visit func(xml.Token) error) error {
	if file == nil {
		return errors.New("Office 본문 XML을 찾을 수 없습니다")
	}
	r, err := file.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	d := xml.NewDecoder(io.LimitReader(r, 32<<20+1))
	depth, tokens := 0, 0
	for {
		if tokens%128 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		t, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.New("Office XML 형식을 읽을 수 없습니다")
		}
		tokens++
		if d.InputOffset() > 32<<20 || tokens > 2000000 {
			return ErrLimit
		}
		switch t.(type) {
		case xml.Directive:
			return errors.New("Office XML 외부 선언은 허용되지 않습니다")
		case xml.StartElement:
			depth++
			if depth > 128 {
				return ErrLimit
			}
		case xml.EndElement:
			depth--
		}
		if err := visit(t); err != nil {
			return err
		}
	}
}

func attr(element xml.StartElement, key string) string {
	for _, a := range element.Attr {
		if a.Name.Local == key {
			return a.Value
		}
	}
	return ""
}

func officeWord(ctx context.Context, files map[string]*zip.File, c *collector) error {
	var text strings.Builder
	paragraph, table, row, column, tableDepth, textDepth := 0, 0, 0, 0, 0, 0
	err := walkXML(ctx, files["word/document.xml"], func(token xml.Token) error {
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				paragraph++
				text.Reset()
			case "tbl":
				tableDepth++
				if tableDepth == 1 {
					table++
					row = 0
				}
			case "tr":
				row++
				column = 0
			case "tc":
				column++
			case "t":
				textDepth++
			case "tab":
				text.WriteByte('\t')
			case "br":
				text.WriteByte('\n')
			case "altChunk":
				c.result.Warnings = appendUnique(c.result.Warnings, "대체 형식 포함 개체는 추출하지 않았습니다")
			}
		case xml.CharData:
			if textDepth > 0 {
				text.Write(t)
				if text.Len() > c.limits.TextBytes {
					return ErrLimit
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				textDepth--
			case "tbl":
				tableDepth--
			case "p":
				p := Position{Paragraph: paragraph}
				if tableDepth > 0 {
					p.Table = table
					p.Row = row
					p.Column = column
				}
				return c.add(text.String(), p)
			}
		}
		return nil
	})
	c.result.Warnings = appendUnique(c.result.Warnings, "Word는 문단·표 위치를 제공합니다. 인쇄 페이지와 머리글·바닥글·주석은 재현하지 않습니다")
	return err
}

func officeRelationships(ctx context.Context, file *zip.File) (map[string]string, error) {
	result := map[string]string{}
	err := walkXML(ctx, file, func(token xml.Token) error {
		if t, ok := token.(xml.StartElement); ok && t.Name.Local == "Relationship" {
			id, target := attr(t, "Id"), attr(t, "Target")
			if strings.EqualFold(attr(t, "TargetMode"), "External") {
				return nil
			}
			if id == "" || target == "" || strings.ContainsAny(target, "\\?#") || strings.HasPrefix(target, "/") || strings.Contains(target, ":") {
				return errors.New("Office 관계 경로를 확인하세요")
			}
			if _, exists := result[id]; exists {
				return errors.New("Office 관계 ID가 중복되었습니다")
			}
			result[id] = target
		}
		return nil
	})
	return result, err
}

func officePresentation(ctx context.Context, files map[string]*zip.File, c *collector) error {
	rels, err := officeRelationships(ctx, files["ppt/_rels/presentation.xml.rels"])
	if err != nil {
		return err
	}
	var slides []string
	err = walkXML(ctx, files["ppt/presentation.xml"], func(token xml.Token) error {
		if t, ok := token.(xml.StartElement); ok && t.Name.Local == "sldId" {
			var id string
			for _, a := range t.Attr {
				if a.Name.Local == "id" && strings.Contains(a.Name.Space, "relationships") {
					id = a.Value
				}
			}
			target, ok := rels[id]
			if !ok {
				return errors.New("슬라이드 연결을 찾을 수 없습니다")
			}
			name := path.Join("ppt", target)
			if !strings.HasPrefix(name, "ppt/slides/") || !strings.HasSuffix(name, ".xml") {
				return errors.New("슬라이드 경로를 확인하세요")
			}
			slides = append(slides, name)
			if len(slides) > c.limits.Pages {
				return ErrLimit
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for index, name := range slides {
		paragraph, textDepth := 0, 0
		var text strings.Builder
		err = walkXML(ctx, files[name], func(token xml.Token) error {
			switch t := token.(type) {
			case xml.StartElement:
				switch t.Name.Local {
				case "p":
					paragraph++
					text.Reset()
				case "t":
					textDepth++
				case "br":
					text.WriteByte('\n')
				}
			case xml.CharData:
				if textDepth > 0 {
					text.Write(t)
					if text.Len() > c.limits.TextBytes {
						return ErrLimit
					}
				}
			case xml.EndElement:
				switch t.Name.Local {
				case "t":
					textDepth--
				case "p":
					return c.add(text.String(), Position{Slide: index + 1, Paragraph: paragraph})
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	c.result.Pages = len(slides)
	c.result.Warnings = appendUnique(c.result.Warnings, "슬라이드의 텍스트 상자·표 텍스트를 추출합니다. 그림·차트·발표자 노트는 포함하지 않습니다")
	return nil
}

var officeCell = regexp.MustCompile(`^[A-Z]{1,3}[1-9][0-9]{0,6}$`)

func officeWorkbook(ctx context.Context, files map[string]*zip.File, c *collector) error {
	rels, err := officeRelationships(ctx, files["xl/_rels/workbook.xml.rels"])
	if err != nil {
		return err
	}
	type sheet struct{ name, path string }
	var sheets []sheet
	err = walkXML(ctx, files["xl/workbook.xml"], func(token xml.Token) error {
		if t, ok := token.(xml.StartElement); ok && t.Name.Local == "sheet" {
			id := attr(t, "id")
			target, ok := rels[id]
			if !ok {
				return errors.New("워크시트 연결을 찾을 수 없습니다")
			}
			name := path.Join("xl", target)
			if !strings.HasPrefix(name, "xl/worksheets/") || !strings.HasSuffix(name, ".xml") {
				return errors.New("워크시트 경로를 확인하세요")
			}
			sheets = append(sheets, sheet{attr(t, "name"), name})
			if len(sheets) > c.limits.Pages {
				return ErrLimit
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	shared := []string{}
	sharedBytes := 0
	if files["xl/sharedStrings.xml"] != nil {
		var text strings.Builder
		textDepth := 0
		err = walkXML(ctx, files["xl/sharedStrings.xml"], func(token xml.Token) error {
			switch t := token.(type) {
			case xml.StartElement:
				if t.Name.Local == "si" {
					text.Reset()
				}
				if t.Name.Local == "t" {
					textDepth++
				}
			case xml.CharData:
				if textDepth > 0 {
					text.Write(t)
					if sharedBytes+text.Len() > c.limits.TextBytes {
						return ErrLimit
					}
				}
			case xml.EndElement:
				if t.Name.Local == "t" {
					textDepth--
				}
				if t.Name.Local == "si" {
					shared = append(shared, text.String())
					sharedBytes += text.Len()
					if len(shared) > 100000 {
						return ErrLimit
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	for _, sheet := range sheets {
		var cell, kind string
		var text strings.Builder
		capture, formula := false, false
		err = walkXML(ctx, files[sheet.path], func(token xml.Token) error {
			switch t := token.(type) {
			case xml.StartElement:
				switch t.Name.Local {
				case "c":
					cell, kind = attr(t, "r"), attr(t, "t")
					text.Reset()
					formula = false
					if !officeCell.MatchString(cell) {
						return errors.New("워크시트 셀 위치가 올바르지 않습니다")
					}
				case "v", "t":
					capture = true
				case "f":
					formula = true
				}
			case xml.CharData:
				if capture {
					text.Write(t)
					if text.Len() > c.limits.TextBytes {
						return ErrLimit
					}
				}
			case xml.EndElement:
				switch t.Name.Local {
				case "v", "t":
					capture = false
				case "c":
					value := text.String()
					if kind == "s" {
						index, e := strconv.Atoi(value)
						if e != nil || index < 0 || index >= len(shared) {
							return errors.New("워크시트 공유 문자열이 올바르지 않습니다")
						}
						value = shared[index]
					}
					if formula {
						c.result.Warnings = appendUnique(c.result.Warnings, "수식은 실행하지 않습니다. 파일에 저장된 마지막 계산 결과만 추출합니다")
					}
					if kind == "b" {
						if value == "1" {
							value = "true"
						} else if value == "0" {
							value = "false"
						}
					}
					return c.add(value, Position{Sheet: sheet.name, Cell: cell})
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	c.result.Pages = len(sheets)
	c.result.Warnings = appendUnique(c.result.Warnings, "셀의 저장 값과 시트·셀 주소를 제공합니다. 사용자 지정 표시 형식·날짜 서식·차트는 재현하지 않습니다")
	sort.Strings(c.result.Warnings)
	return nil
}
