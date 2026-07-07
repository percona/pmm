import { CodeBlock } from '@percona/percona-ui';
import { FC } from 'react';

export interface Props {
  query: string;
}

const QueryCell: FC<Props> = ({ query }) => (
  <CodeBlock
    content={query
      .replace(/[\n\r\t]/g, '')
      .replace(/\s{2,}/g, ' ')
      .trim()}
    language="javascript"
    sx={{ width: '100%' }}
  />
);

export default QueryCell;
