import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockFrame, BlockMarkdown } from './BlockFrame';

export const SummaryBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = block.dataJson as { summary?: string } | undefined;
  const text = data?.summary ?? block.title ?? '';
  return (
    <BlockFrame title={block.title}>
      <BlockMarkdown text={text} />
    </BlockFrame>
  );
};
