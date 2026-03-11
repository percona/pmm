import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { MarkdownBlock } from './MarkdownBlock';
import { SummaryBlock } from './SummaryBlock';

export const BlockRenderer: FC<{ block: InvestigationBlock }> = ({ block }) => {
  switch (block.type) {
    case 'summary':
      return <SummaryBlock block={block} />;
    case 'markdown':
      return <MarkdownBlock block={block} />;
    default:
      return (
        <MarkdownBlock
          block={{
            ...block,
            type: 'markdown',
            dataJson: block.dataJson ?? { content: `(Unsupported block type: ${block.type})` },
          }}
        />
      );
  }
};
