import { CodeBlock } from 'components/code-block';
import { FC } from 'react';

export interface Props {
  query: string;
}

const QueryCell: FC<Props> = ({ query }) => (
  <CodeBlock
    code={query.replace(/\n/g, '').replace(/  /g, '')}
    containerProps={{
      sx: {
        width: '100%',
      },
    }}
    language="mongodb"
  />
);

export default QueryCell;
