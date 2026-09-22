package server

// Diagnostic/Agent graph views must not materialize 100 complete 4MiB bodies
// or arbitrary-size alias arrays. Report every omitted source/alias explicitly;
// preserve accepted alias values rather than clipping them into new names.
const limitedGraphDocumentJSON = `jsonb_build_object('id',d.id,'title',left(d.title,500),'markdown',left(d.markdown,32000),
 'aliases',` + documentSummaryAliases + `,
 'source_truncated',char_length(d.markdown)>32000,
 'aliases_truncated',CASE WHEN jsonb_typeof(d.aliases)='array' THEN jsonb_array_length(d.aliases)>jsonb_array_length(` + documentSummaryAliases + `) ELSE true END)`
