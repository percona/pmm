import { FC, ReactNode } from 'react';
import Accordion from '@mui/material/Accordion';
import AccordionDetails from '@mui/material/AccordionDetails';
import AccordionSummary from '@mui/material/AccordionSummary';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { CodeBlock } from '@percona/percona-ui';
import { Template } from 'types/alert-templates.types';
import { Messages } from '../CreateAlertFromTemplate.messages';

interface Props {
  template: Template | null;
}

interface BlockProps {
  title: string;
  testId: string;
  children: ReactNode;
}

const Block: FC<BlockProps> = ({ title, testId, children }) => (
  <Stack gap={1} data-testid={testId}>
    <Typography variant="subtitle2">{title}</Typography>
    {children}
  </Stack>
);

// One query/expression step of a multi-expression template, labelled by its ref ID.
const Step: FC<{ refId: string; value: string; testId: string }> = ({
  refId,
  value,
  testId,
}) => (
  <Stack gap={0.5}>
    <Typography variant="caption" color="text.secondary">
      {`${refId}:`}
    </Typography>
    <CodeBlock content={value} copyable wrap data-testid={testId} />
  </Stack>
);

// Read-only view of the template's underlying queries. Params are still shown as
// raw `[[ .placeholder ]]` markers; the server substitutes them when the rule is
// created, so there is nothing to interpolate here.
export const AdvancedSection: FC<Props> = ({ template }) => {
  if (!template) {
    return null;
  }

  const { queries = [], expressions = [], condition } = template;
  const summary = template.annotations?.summary;

  return (
    <Accordion disableGutters data-testid="alert-rule-advanced-section">
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Typography variant="h6">{Messages.sections.advanced}</Typography>
      </AccordionSummary>
      <AccordionDetails>
        <Stack gap={2}>
          {queries.length > 0 && (
            <Block title={Messages.advanced.queries} testId="template-queries">
              {queries.map((query) => (
                <Step
                  key={query.refId}
                  refId={query.refId}
                  value={query.expr}
                  testId={`template-query-${query.refId}`}
                />
              ))}
            </Block>
          )}
          {expressions.length > 0 && (
            <Block
              title={Messages.advanced.expressions}
              testId="template-expressions"
            >
              {expressions.map((expression) => (
                <Step
                  key={expression.refId}
                  refId={expression.refId}
                  value={expression.expression}
                  testId={`template-expression-${expression.refId}`}
                />
              ))}
            </Block>
          )}
          {condition && (
            <Block
              title={Messages.advanced.condition}
              testId="template-condition"
            >
              <CodeBlock content={condition} />
            </Block>
          )}
          {/* Single-expression templates only. Gated on `queries` rather than
              `expressions` because the backend rejects a template that sets both
              `expr` and `queries`, but allows `queries` with no `expressions`. */}
          {queries.length === 0 && (
            <Block
              title={Messages.advanced.expression}
              testId="template-expression"
            >
              <CodeBlock content={template.expr} copyable wrap />
            </Block>
          )}
          {summary && (
            <Block title={Messages.advanced.ruleAlert} testId="template-alert">
              <CodeBlock content={summary} wrap />
            </Block>
          )}
        </Stack>
      </AccordionDetails>
    </Accordion>
  );
};

export default AdvancedSection;
