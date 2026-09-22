package server

import (
	"bytes"
	"context"
	"mime"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type ProtectionFileResult struct {
	Name        string
	Data        []byte
	Findings    []ProtectionFinding
	Changed     bool
	Unscannable bool
}

// ProtectAttachmentTx scans only explicitly supported UTF-8 textual files. It
// never pretends that images, PDFs, archives, or other binary formats were read.
func (s *Server) ProtectAttachmentTx(ctx context.Context, tx pgx.Tx, p *Principal, documentID, workspaceID, name, contentType string, data []byte) (ProtectionFileResult, error) {
	result := ProtectionFileResult{Name: name, Data: data, Findings: []ProtectionFinding{}}
	var raw []byte
	if e := tx.QueryRow(ctx, "SELECT data FROM protection_settings WHERE id=1 FOR SHARE").Scan(&raw); e != nil {
		return result, e
	}
	cfg, e := decodeProtectionSettings(raw)
	if e != nil {
		return result, e
	}
	if !boolean(cfg, "enabled") {
		return result, nil
	}
	mode := str(cfg, "mode")
	cleanedName, findings, e := protectionScanTx(ctx, tx, cfg, name)
	if e != nil {
		return result, e
	}
	result.Findings = append(result.Findings, findings...)
	kind, _, _ := mime.ParseMediaType(contentType)
	supported := (strings.HasPrefix(kind, "text/") || oneOf(kind, "application/json", "application/xml", "application/yaml", "application/x-yaml", "application/javascript") || strings.HasSuffix(kind, "+json") || strings.HasSuffix(kind, "+xml")) && len(data) <= 4<<20 && utf8.Valid(data) && bytes.IndexByte(data, 0) < 0
	cleaned := string(data)
	if supported {
		var f []ProtectionFinding
		cleaned, f, e = protectionScanTx(ctx, tx, cfg, string(data))
		if e != nil {
			return result, e
		}
		result.Findings = append(result.Findings, f...)
	} else {
		result.Unscannable = true
		if str(cfg, "unscannable") == "block" {
			return result, ProtectionError{Code: "unscannable", Findings: []ProtectionFinding{{Kind: "unscannable", Count: 1}}}
		}
	}
	if mode == "block" && len(result.Findings) > 0 {
		return result, ProtectionError{Code: "blocked", Findings: result.Findings}
	}
	if mode == "mask" && len(result.Findings) > 0 {
		result.Name = cleanedName
		if supported {
			result.Data = []byte(cleaned)
		}
		result.Changed = true
	}
	if len(result.Findings) > 0 || result.Unscannable {
		recorded := append([]ProtectionFinding{}, result.Findings...)
		if result.Unscannable {
			recorded = append(recorded, ProtectionFinding{Kind: "unscannable", Count: 1})
		}
		actor := ""
		if p != nil {
			actor = p.ID
		}
		_, e = tx.Exec(ctx, "INSERT INTO protection_events(id,document_id,workspace_id,user_id,action,mode,findings) VALUES($1,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,'attachment.scan',$5,$6)", newID(), documentID, workspaceID, actor, mode, jsonValue(recorded))
	}
	return result, e
}
