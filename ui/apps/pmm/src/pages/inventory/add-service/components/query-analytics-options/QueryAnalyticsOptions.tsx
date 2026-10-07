import type { FC } from 'react';
import { SwitchInput } from '@percona/peak-ui';
import { Messages } from './QueryAnalyticsOptions.messages';

export const QueryAnalyticsOptions: FC = () => (
  <>
    <SwitchInput
      name="disableQueryExamples"
      label={Messages.disableQueryExamples}
    />
    <SwitchInput
      name="disableCommentsParsing"
      label={Messages.disableCommentsParsing}
    />
  </>
);
