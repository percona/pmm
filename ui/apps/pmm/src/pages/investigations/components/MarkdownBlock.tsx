import { Card, CardContent, Typography } from '@mui/material';
import { FC, useMemo } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import type { InvestigationBlock } from 'api/investigations';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';

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
  const isLog = useMemo(() => isLogBlock(block.title, rawContent), [block.title, rawContent]);
  const content = useMemo(
    () => (isLog ? sortLogLinesOldestFirst(rawContent) : rawContent),
    [isLog, rawContent]
  );
  return (
    <Card variant="outlined" sx={{ mb: 2, bgcolor: 'action.hover', borderLeft: 4, borderLeftColor: 'grey.400' }}>
      {block.title && (
        <CardContent sx={{ pb: 0 }}>
          <Typography variant="subtitle1" fontWeight={600}>
            {block.title}
          </Typography>
        </CardContent>
      )}
      <CardContent>
        {isLog ? (
          // Logs are line-oriented: render verbatim in a monospace block so each entry stays on
          // its own line (Markdown would collapse the single newlines into spaces).
          <Typography
            component="pre"
            variant="body2"
            sx={{
              fontFamily: 'Roboto Mono, monospace',
              fontSize: '0.8rem',
              lineHeight: 1.5,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              overflowX: 'auto',
              m: 0,
            }}
          >
            {content}
          </Typography>
        ) : (
          <Typography component="div" variant="body2">
            <Markdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw]}
              components={getMarkdownComponents(content)}
            >
              {content}
            </Markdown>
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};
