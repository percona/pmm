import { type FC, useMemo } from 'react';
import { Stack, Typography } from '@mui/material';
import { CodeBlock } from '@percona/peak-ui';
import { Messages } from '../AlertDetailsTab.messages';
import { RULE_STEP_LABELS } from './RuleExpression.constants';
import { getRuleSteps } from './RuleExpression.utils';
import { RuleExpressionProps } from './RuleExpression.types';

const RuleExpression: FC<RuleExpressionProps> = ({ definition, fallback }) => {
  const steps = useMemo(
    () => (definition ? getRuleSteps(definition) : []),
    [definition]
  );

  if (!steps.length) {
    return <CodeBlock language="promql" content={fallback} />;
  }

  return (
    <Stack spacing={2}>
      {steps.map(({ refId, kind, content, language, isCondition }) => (
        <Stack key={refId} spacing={0.5} data-testid={`rule-step-${refId}`}>
          <Typography variant="subtitle2">
            {[
              refId,
              RULE_STEP_LABELS[kind],
              isCondition && Messages.details.stepCondition,
            ]
              .filter(Boolean)
              .join(' · ')}
          </Typography>
          <CodeBlock language={language} content={content} />
        </Stack>
      ))}
    </Stack>
  );
};

export default RuleExpression;
