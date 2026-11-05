import type { FC } from 'react';
import { SwitchInput } from '@percona/peak-ui';
import { Messages } from './QueryAnalyticsOptions.messages';
import type { QueryAnalyticsOptionsProps } from './QueryAnalyticsOptions.types';

export const QueryAnalyticsOptions: FC<QueryAnalyticsOptionsProps> = ({
  withCommentsParsing = true,
}) => (
  <>
    <SwitchInput
      name="disableQueryExamples"
      label={Messages.disableQueryExamples}
    />
    {withCommentsParsing && (
      <SwitchInput
        name="disableCommentsParsing"
        label={Messages.disableCommentsParsing}
      />
    )}
  </>
);
