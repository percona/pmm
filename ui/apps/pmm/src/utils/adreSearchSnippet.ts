/**
 * PostgreSQL ts_headline uses StartSel=<< / StopSel=>> in our search SQL; strip for readable UI.
 */
export function formatAdreSearchSnippet(snippet: string): string {
  if (!snippet) return '';
  return snippet.replace(/<<|>>/g, '').trim();
}
