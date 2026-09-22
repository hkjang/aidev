package extract

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"image"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type runner func(context.Context, string, string, int) ([]byte, error)

func PDF(ctx context.Context, dir string, ocrPages []int, limits Limits) (Result, error) {
	return readPDF(ctx, dir, ocrPages, limits, Run)
}

func readPDF(ctx context.Context, dir string, ocrPages []int, limits Limits, run runner) (Result, error) {
	c := newCollector(limits)
	info, err := run(ctx, dir, "pdf-info", 0)
	if err != nil {
		return c.result, err
	}
	pages, err := pdfPages(info)
	if err != nil {
		return c.result, err
	}
	if pages < 1 || pages > limits.Pages {
		return c.result, ErrLimit
	}
	selected := map[int]bool{}
	for _, page := range ocrPages {
		if page < 1 || page > pages || selected[page] {
			return c.result, errors.New("OCR 페이지 선택을 확인하세요")
		}
		selected[page] = true
	}
	if len(selected) > 20 {
		return c.result, errors.New("한 번에 OCR할 빈 페이지는 최대 20개입니다")
	}
	c.result.Pages = pages
	for page := 1; page <= pages; page++ {
		if err := ctx.Err(); err != nil {
			return c.result, err
		}
		data, err := run(ctx, dir, "pdf-text", page)
		if err != nil {
			return c.result, err
		}
		before := len(c.result.Fragments)
		size, err := pdfWords(data, page, c)
		if err != nil {
			return c.result, err
		}
		if len(c.result.Fragments) > before {
			if selected[page] {
				c.result.Warnings = appendUnique(c.result.Warnings, "이미 텍스트가 있는 페이지는 OCR하지 않았습니다")
			}
			continue
		}
		c.result.EmptyPages = append(c.result.EmptyPages, page)
		if !selected[page] {
			continue
		}
		if _, err = run(ctx, dir, "pdf-render", page); err != nil {
			return c.result, err
		}
		filename := filepath.Join(dir, "page.png")
		f, err := os.Open(filename)
		if err != nil {
			return c.result, err
		}
		img, _, err := image.DecodeConfig(f)
		f.Close()
		if err != nil || img.Width < 1 || img.Height < 1 || img.Width > 2400 || img.Height > 2400 {
			return c.result, errors.New("OCR 페이지 이미지 크기를 확인하세요")
		}
		data, err = run(ctx, dir, "ocr", page)
		removeErr := os.Remove(filename)
		if err != nil {
			return c.result, err
		}
		if removeErr != nil {
			return c.result, removeErr
		}
		if err = ocrWords(data, page, size, img.Width, img.Height, c); err != nil {
			return c.result, err
		}
		c.result.Warnings = appendUnique(c.result.Warnings, "OCR은 선택한 빈 페이지에만 실행했습니다. 인식 오류가 있을 수 있으므로 원본 페이지와 대조하세요")
	}
	c.result.Warnings = appendUnique(c.result.Warnings, "PDF의 실제 텍스트·페이지·영역 좌표를 보존합니다. 다단 편집·표의 읽기 순서는 원본 화면과 다를 수 있습니다")
	return c.result, nil
}

func pdfPages(data []byte) (int, error) {
	pages := 0
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "Pages":
			pages, _ = strconv.Atoi(value)
		case "Encrypted":
			if strings.HasPrefix(value, "yes") {
				return 0, errors.New("암호화된 PDF는 추출하지 않습니다")
			}
		}
	}
	if pages < 1 {
		return 0, errors.New("PDF 페이지 수를 읽을 수 없습니다")
	}
	return pages, nil
}

func numberAttr(e xml.StartElement, name string) (float64, error) {
	v, err := strconv.ParseFloat(attr(e, name), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < -1e7 || v > 1e7 {
		return 0, errors.New("PDF 위치 값이 올바르지 않습니다")
	}
	return v, nil
}

func pdfWords(data []byte, page int, c *collector) ([]float64, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	size := []float64{0, 0}
	var text, word strings.Builder
	var bounds []float64
	inWord := false
	depth := 0
	for {
		token, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return size, errors.New("PDF 텍스트 위치를 읽을 수 없습니다")
		}
		switch t := token.(type) {
		case xml.Directive:
			if strings.Contains(strings.ToUpper(string(t)), "ENTITY") {
				return size, errors.New("PDF 위치 XML의 외부 선언은 허용하지 않습니다")
			}
		case xml.StartElement:
			depth++
			if depth > 64 {
				return size, ErrLimit
			}
			switch t.Name.Local {
			case "page":
				size[0], e = numberAttr(t, "width")
				if e == nil {
					size[1], e = numberAttr(t, "height")
				}
				if e != nil || size[0] <= 0 || size[1] <= 0 {
					return size, errors.New("PDF 페이지 크기를 확인하세요")
				}
			case "line":
				text.Reset()
				bounds = []float64{}
				for _, key := range []string{"xMin", "yMin", "xMax", "yMax"} {
					n, err := numberAttr(t, key)
					if err != nil {
						return size, err
					}
					bounds = append(bounds, n)
				}
			case "word":
				inWord = true
				word.Reset()
			}
		case xml.CharData:
			if inWord {
				word.Write(t)
				if word.Len() > 16<<10 {
					return size, ErrLimit
				}
			}
		case xml.EndElement:
			depth--
			switch t.Name.Local {
			case "word":
				inWord = false
				if text.Len() > 0 {
					text.WriteByte(' ')
				}
				text.WriteString(word.String())
				if text.Len() > c.limits.TextBytes {
					return size, ErrLimit
				}
			case "line":
				if len(bounds) != 4 || bounds[2] < bounds[0] || bounds[3] < bounds[1] {
					return size, errors.New("PDF 텍스트 영역이 페이지를 벗어났습니다")
				}
				bounds = []float64{math.Max(0, bounds[0]), math.Max(0, bounds[1]), math.Min(size[0], bounds[2]), math.Min(size[1], bounds[3])}
				if bounds[2] <= bounds[0] || bounds[3] <= bounds[1] {
					continue
				}
				if e = c.add(text.String(), Position{Page: page, Bounds: bounds, PageSize: append([]float64{}, size...)}); e != nil {
					return size, e
				}
			}
		}
	}
	if size[0] <= 0 || size[1] <= 0 {
		return size, errors.New("PDF 페이지 위치 정보를 찾을 수 없습니다")
	}
	return size, nil
}

func ocrWords(data []byte, page int, size []float64, width, height int, c *collector) error {
	r := bufio.NewScanner(bytes.NewReader(data))
	r.Buffer(make([]byte, 4096), 64<<10)
	if !r.Scan() || r.Text() != "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext" {
		return errors.New("OCR 위치 결과를 읽을 수 없습니다")
	}
	var key string
	var text strings.Builder
	var bounds []float64
	flush := func() error {
		if text.Len() == 0 {
			return nil
		}
		p := Position{Page: page, OCR: true, Bounds: bounds, PageSize: append([]float64{}, size...)}
		if len(size) == 2 && size[0] > 0 && size[1] > 0 {
			p.Bounds = []float64{bounds[0] * size[0] / float64(width), bounds[1] * size[1] / float64(height), bounds[2] * size[0] / float64(width), bounds[3] * size[1] / float64(height)}
		}
		return c.add(text.String(), p)
	}
	for r.Scan() {
		row := strings.SplitN(r.Text(), "\t", 12)
		if len(row) != 12 {
			return errors.New("OCR 위치 행이 올바르지 않습니다")
		}
		if row[0] != "5" || strings.TrimSpace(row[11]) == "" {
			continue
		}
		next := strings.Join(row[1:5], "/")
		if next != key {
			if e := flush(); e != nil {
				return e
			}
			key = next
			text.Reset()
			bounds = nil
		}
		numbers := []float64{}
		for _, raw := range row[6:10] {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 || n > 2400 {
				return errors.New("OCR 영역 크기를 확인하세요")
			}
			numbers = append(numbers, float64(n))
		}
		b := []float64{numbers[0], numbers[1], numbers[0] + numbers[2], numbers[1] + numbers[3]}
		if b[2] > float64(width) || b[3] > float64(height) {
			return errors.New("OCR 영역이 이미지를 벗어났습니다")
		}
		if bounds == nil {
			bounds = b
		} else {
			bounds = []float64{math.Min(bounds[0], b[0]), math.Min(bounds[1], b[1]), math.Max(bounds[2], b[2]), math.Max(bounds[3], b[3])}
		}
		if text.Len() > 0 {
			text.WriteByte(' ')
		}
		text.WriteString(row[11])
		if text.Len() > c.limits.TextBytes {
			return ErrLimit
		}
	}
	if e := r.Err(); e != nil {
		return ErrLimit
	}
	return flush()
}
