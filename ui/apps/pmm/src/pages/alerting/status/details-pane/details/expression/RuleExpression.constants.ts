import { Messages } from '../AlertDetailsTab.messages';
import { RuleStepKind } from './RuleExpression.utils';

export const RULE_STEP_LABELS: Record<RuleStepKind, string> = {
  [RuleStepKind.Query]: Messages.details.stepQuery,
  [RuleStepKind.SqlQuery]: Messages.details.stepSqlQuery,
  [RuleStepKind.Threshold]: Messages.details.stepThreshold,
  [RuleStepKind.Reduce]: Messages.details.stepReduce,
  [RuleStepKind.Math]: Messages.details.stepMath,
  [RuleStepKind.Expression]: Messages.details.stepExpression,
};
