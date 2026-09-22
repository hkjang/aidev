package server

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
)

const inboundCaptureMax = 10 << 20

type inboundAttachment struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data []byte `json:"data"`
}
type inboundContent struct {
	Title       string              `json:"title"`
	Markdown    string              `json:"markdown"`
	MessageID   string              `json:"message_id"`
	Source      map[string]any      `json:"source"`
	Attachments []inboundAttachment `json:"attachments"`
}

func inboundDecodeText(raw []byte, encoding string) (string, error) {
	if encoding != "" && !strings.EqualFold(encoding, "utf-8") && !strings.EqualFold(encoding, "us-ascii") {
		reader, e := charset.NewReaderLabel(encoding, bytes.NewReader(raw))
		if e != nil {
			return "", errors.New("지원하지 않는 메일 문자 인코딩입니다")
		}
		raw, e = io.ReadAll(io.LimitReader(reader, 4<<20+1))
		if e != nil {
			return "", e
		}
	}
	if len(raw) > 4<<20 || !utf8.Valid(raw) {
		return "", errors.New("메일 본문은 UTF-8로 변환 가능한 4MB 이하여야 합니다")
	}
	return string(raw), nil
}
func inboundFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == ".." {
		name = "첨부파일"
	}
	runes := []rune(name)
	if len(runes) > 180 {
		name = string(runes[:180])
	}
	return name
}
func parseInboundMIME(raw []byte) (inboundContent, error) {
	out := inboundContent{Source: map[string]any{"origin": "email"}, Attachments: []inboundAttachment{}}
	if len(raw) > inboundCaptureMax {
		return out, errors.New("수집 메일은 MIME 원문 10MB 이하여야 합니다")
	}
	message, e := mail.ReadMessage(bytes.NewReader(raw))
	if e != nil {
		return out, errors.New("메일 헤더 형식을 읽을 수 없습니다")
	}
	decoder := &mime.WordDecoder{CharsetReader: func(label string, input io.Reader) (io.Reader, error) { return charset.NewReaderLabel(label, input) }}
	out.Title, e = decoder.DecodeHeader(message.Header.Get("Subject"))
	if e != nil {
		return out, errors.New("메일 제목을 해독할 수 없습니다")
	}
	out.Title = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(out.Title, "\r", " "), "\n", " "))
	if out.Title == "" {
		out.Title = "수집한 이메일"
	}
	if len([]rune(out.Title)) > 150 {
		out.Title = string([]rune(out.Title)[:150])
	}
	out.MessageID = strings.TrimSpace(message.Header.Get("Message-ID"))
	if len(out.MessageID) > 998 || strings.ContainsAny(out.MessageID, "\r\n") {
		return out, errors.New("메일 Message-ID 형식을 확인하세요")
	}
	out.Source["message_id"] = out.MessageID
	if len(message.Header.Get("From")) > 4096 || len(message.Header.Get("Date")) > 256 {
		return out, errors.New("메일 발신자 또는 날짜 헤더가 너무 큽니다")
	}
	out.Source["from"] = message.Header.Get("From")
	out.Source["date"] = message.Header.Get("Date")
	plain, htmlParts := []string{}, []string{}
	parts, total := 0, 0
	var visit func(textproto.MIMEHeader, io.Reader, int) error
	visit = func(header textproto.MIMEHeader, body io.Reader, depth int) error {
		parts++
		if parts > 100 || depth > 10 {
			return errors.New("메일 MIME 계층 또는 파트 수가 너무 큽니다")
		}
		media, params, e := mime.ParseMediaType(header.Get("Content-Type"))
		if header.Get("Content-Type") == "" {
			media = "text/plain"
			params = map[string]string{}
		} else if e != nil {
			return errors.New("메일 Content-Type을 확인하세요")
		}
		if strings.HasPrefix(media, "multipart/") {
			boundary := params["boundary"]
			if boundary == "" {
				return errors.New("메일 multipart 경계가 없습니다")
			}
			reader := multipart.NewReader(body, boundary)
			for {
				part, e := reader.NextRawPart()
				if e == io.EOF {
					return nil
				}
				if e != nil {
					return errors.New("메일 multipart를 읽을 수 없습니다")
				}
				e = visit(part.Header, part, depth+1)
				part.Close()
				if e != nil {
					return e
				}
			}
		}
		switch strings.ToLower(strings.TrimSpace(header.Get("Content-Transfer-Encoding"))) {
		case "base64":
			body = base64.NewDecoder(base64.StdEncoding, body)
		case "quoted-printable":
			body = quotedprintable.NewReader(body)
		case "", "7bit", "8bit", "binary":
		default:
			return errors.New("지원하지 않는 MIME 전송 인코딩입니다")
		}
		data, e := io.ReadAll(io.LimitReader(body, inboundCaptureMax+1))
		if e != nil {
			return errors.New("메일 파트 인코딩을 읽을 수 없습니다")
		}
		total += len(data)
		if len(data) > inboundCaptureMax || total > inboundCaptureMax {
			return errors.New("메일 해제 크기는 10MB 이하여야 합니다")
		}
		disposition, dparams, _ := mime.ParseMediaType(header.Get("Content-Disposition"))
		name := dparams["filename"]
		if name == "" {
			name = params["name"]
		}
		attachment := name != "" || disposition == "attachment" || !oneOf(media, "text/plain", "text/html")
		if attachment {
			if len(out.Attachments) >= 20 {
				return errors.New("메일 첨부는 최대 20개입니다")
			}
			name, e = decoder.DecodeHeader(name)
			if e != nil {
				return errors.New("첨부파일 이름을 해독할 수 없습니다")
			}
			out.Attachments = append(out.Attachments, inboundAttachment{Name: inboundFilename(name), Type: media, Data: data})
			return nil
		}
		text, e := inboundDecodeText(data, params["charset"])
		if e != nil {
			return e
		}
		if media == "text/html" {
			_, markdown, e := migrationHTML([]byte(text))
			if e != nil {
				return e
			}
			htmlParts = append(htmlParts, markdown)
		} else {
			plain = append(plain, text)
		}
		return nil
	}
	if e = visit(textproto.MIMEHeader(message.Header), message.Body, 0); e != nil {
		return out, e
	}
	if len(plain) > 0 {
		out.Markdown = strings.Join(plain, "\n\n")
	} else {
		out.Markdown = strings.Join(htmlParts, "\n\n")
	}
	if len(out.Markdown) > 4<<20 {
		return out, errors.New("메일 본문은 4MB 이하여야 합니다")
	}
	return out, nil
}
