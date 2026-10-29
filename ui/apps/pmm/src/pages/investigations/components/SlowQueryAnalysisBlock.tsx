import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockEmpty, BlockFrame, BlockMarkdown, BlockPre } from './BlockFrame';

export const SlowQueryAnalysisBlock: FC<{ block: InvestigationBlock }> = ({
  block,
}) => {
  const data = (block.dataJson || {}) as {
    content?: string;
    summary?: string;
    query?: string;
  };
  const text = data.content ?? data.summary ?? block.title ?? '';
  return (
    <BlockFrame title={block.title}>
      {data.query && <BlockPre caption>{data.query}</BlockPre>}
      {text ? (
        <BlockMarkdown text={text} />
      ) : (
        <BlockEmpty>(No analysis)</BlockEmpty>
      )}
    </BlockFrame>
  );
};
