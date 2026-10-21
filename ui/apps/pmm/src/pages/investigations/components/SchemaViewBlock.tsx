import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockFrame, BlockPre } from './BlockFrame';

export const SchemaViewBlock: FC<{ block: InvestigationBlock }> = ({
  block,
}) => {
  const data = (block.dataJson || {}) as { content?: string; schema?: string };
  const text = data.content ?? data.schema ?? block.title ?? '';
  return (
    <BlockFrame title={block.title}>
      <BlockPre>{text || '(No schema)'}</BlockPre>
    </BlockFrame>
  );
};
