package server

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type canonicalDocumentProtection struct {
	ProtectionResult
	Tags, Aliases []string
	Metadata      any
}

// Apply protection before snapshots, task projections and events are produced.
// If the source changes, editor metadata cannot safely keep source-bound IDs or
// hidden copies of the original text, so the editor will derive it afresh.
func (s *Server) protectCanonicalDocumentTx(ctx context.Context, tx pgx.Tx, p *Principal, id, wid, title, markdown string, tags, aliases []string, metadata any) (canonicalDocumentProtection, error) {
	result, err := s.ProtectDocumentTx(ctx, tx, p, id, wid, title, markdown)
	out := canonicalDocumentProtection{ProtectionResult: result, Tags: tags, Aliases: aliases, Metadata: metadata}
	if err != nil {
		return out, err
	}
	if result.Markdown != markdown {
		metadata = map[string]any{}
	}
	meta, err := s.ProtectDocumentMetadataTx(ctx, tx, p, id, wid, map[string]any{"tags": tags, "aliases": aliases, "block_metadata": metadata})
	if err != nil {
		return out, err
	}
	fields := meta.Value.(map[string]any)
	out.Tags, out.Aliases, out.Metadata = listStrings(fields["tags"]), listStrings(fields["aliases"]), fields["block_metadata"]
	out.Changed = out.Changed || meta.Changed
	out.Findings = append(out.Findings, meta.Findings...)
	if len(out.Title) == 0 || len(out.Title) > 500 || len(out.Markdown) > 4<<20 || len(jsonValue(out.Metadata)) > 1<<20 {
		return out, ProtectionError{Code: "mask_required", Findings: out.Findings}
	}
	return out, nil
}

func (p canonicalDocumentProtection) response() map[string]any {
	return map[string]any{"changed": p.Changed, "mode": p.Mode, "findings": p.Findings}
}
