import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockFrame, BlockPre } from './BlockFrame';

export const LogsViewBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { content?: string; lines?: string[] };
  const text =
    data.content ??
    (Array.isArray(data.lines) ? data.lines.join('\n') : '') ??
    block.title ??
    '';
  return (
    <BlockFrame title={block.title}>
      <BlockPre>{text || '(No logs)'}</BlockPre>
    </BlockFrame>
  );
};
