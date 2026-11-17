import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockEmpty, BlockFrame, BlockMarkdown } from './BlockFrame';

export const FindingBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { content?: string; summary?: string };
  const text = data.content ?? data.summary ?? block.title ?? '';
  return (
    <BlockFrame title={block.title}>
      {text ? (
        <BlockMarkdown text={text} />
      ) : (
        <BlockEmpty>(No content)</BlockEmpty>
      )}
    </BlockFrame>
  );
};
