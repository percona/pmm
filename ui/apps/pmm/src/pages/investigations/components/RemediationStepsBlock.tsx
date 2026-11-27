import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { BlockEmpty, BlockFrame, BlockMarkdown } from './BlockFrame';
import { stepsMarkdown } from './remediationSteps.utils';

export const RemediationStepsBlock: FC<{ block: InvestigationBlock }> = ({
  block,
}) => {
  const data = (block.dataJson || {}) as { steps?: string[]; content?: string };
  const combined = (() => {
    if (typeof data.content === 'string' && data.content.trim()) {
      return data.content;
    }
    if (Array.isArray(data.steps) && data.steps.length > 0) {
      return stepsMarkdown(
        data.steps.filter((step): step is string => typeof step === 'string')
      );
    }
    return '';
  })();
  return (
    <BlockFrame title={block.title}>
      {combined ? (
        <BlockMarkdown text={combined} />
      ) : (
        <BlockEmpty>(No steps)</BlockEmpty>
      )}
    </BlockFrame>
  );
};
