export type EditingSnapshot = { title: string; markdown: string; tags: string };

// A save response may normalize or redact content. Only replace fields whose
// current draft still equals the submitted value; typing after send must win.
export function reconcileSavedDocument(
  submitted: EditingSnapshot,
  latest: EditingSnapshot,
  canonical: { title: string; markdown: string; tags: string[] },
  pendingCollaboration = false,
) {
  const applyMarkdown =
    !pendingCollaboration && latest.markdown === submitted.markdown;
  const values = {
    title: latest.title === submitted.title ? canonical.title : latest.title,
    tags:
      latest.tags === submitted.tags ? canonical.tags.join(", ") : latest.tags,
    markdown: applyMarkdown ? canonical.markdown : latest.markdown,
  };
  const dirty =
    pendingCollaboration ||
    values.title !== canonical.title ||
    values.tags !== canonical.tags.join(", ") ||
    values.markdown !== canonical.markdown;
  return { values, applyMetadata: applyMarkdown, dirty };
}
