import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockFrame, BlockPre } from './BlockFrame';

export const TopQueriesBlock: FC<{ block: InvestigationBlock }> = ({
  block,
}) => {
  const data = (block.dataJson || {}) as {
    content?: string;
    queries?: unknown[];
  };
  const text =
    data.content ??
    (Array.isArray(data.queries)
      ? JSON.stringify(data.queries, null, 2)
      : '') ??
    block.title ??
    '';
  return (
    <BlockFrame title={block.title}>
      <BlockPre>{text || '(No queries)'}</BlockPre>
    </BlockFrame>
  );
};
