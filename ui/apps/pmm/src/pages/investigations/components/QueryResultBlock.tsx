import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockFrame, BlockPre } from './BlockFrame';

export const QueryResultBlock: FC<{ block: InvestigationBlock }> = ({
  block,
}) => {
  const data = (block.dataJson || {}) as {
    query?: string;
    result?: string;
    rows?: unknown[];
  };
  const text =
    data.result ??
    (Array.isArray(data.rows) ? JSON.stringify(data.rows, null, 2) : '') ??
    block.title ??
    '';
  return (
    <BlockFrame title={block.title}>
      {data.query && <BlockPre caption>{data.query}</BlockPre>}
      <BlockPre>{text || '(No result)'}</BlockPre>
    </BlockFrame>
  );
};
