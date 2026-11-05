import { FC, useMemo } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockFrame, BlockMarkdown, BlockPre } from './BlockFrame';

const LOG_TIMESTAMP_RE = /^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(?:\.\d+)?)\s/;

function sortLogLinesOldestFirst(text: string): string {
  const lines = text.split('\n');
  const withTimestamp: Array<{ line: string; ts: string }> = [];
  const withoutTimestamp: string[] = [];
  for (const line of lines) {
    const m = line.match(LOG_TIMESTAMP_RE);
    if (m) {
      withTimestamp.push({ line, ts: m[1] });
    } else {
      withoutTimestamp.push(line);
    }
  }
  withTimestamp.sort((a, b) => a.ts.localeCompare(b.ts));
  const sorted = [
    ...withoutTimestamp,
    ...withTimestamp.map(({ line }) => line),
  ];
  return sorted.join('\n');
}

function isLogBlock(title?: string, content?: string): boolean {
  if (!content) return false;
  const t = (title ?? '').toLowerCase();
  if (t.includes('related logs') || t.includes('logs from')) return true;
  return LOG_TIMESTAMP_RE.test(content);
}

export const MarkdownBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = block.dataJson as { content?: string } | undefined;
  const rawContent = data?.content ?? '';
  const isLog = useMemo(
    () => isLogBlock(block.title, rawContent),
    [block.title, rawContent]
  );
  const content = useMemo(
    () => (isLog ? sortLogLinesOldestFirst(rawContent) : rawContent),
    [isLog, rawContent]
  );
  return (
    <BlockFrame title={block.title}>
      {isLog ? (
        // Logs are line-oriented: keep each entry on its own line (Markdown would join single newlines).
        <BlockPre>{content}</BlockPre>
      ) : (
        <BlockMarkdown text={content} />
      )}
    </BlockFrame>
  );
};
