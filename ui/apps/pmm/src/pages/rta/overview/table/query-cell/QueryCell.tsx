import { CodeBlock } from 'components/code-block';
import { FC } from 'react';

export interface Props {
  query: string;
}

const QueryCell: FC<Props> = ({ query }) => (
  <CodeBlock
    code={query
      .replace(/[\n\r\t]/g, '')
      .replace(/\s{2,}/g, ' ')
      .trim()}
    containerProps={{
      sx: {
        width: '100%',
        maxHeight: '50px',
      },
    }}
    language="mongodb"
  />
);

export default QueryCell;
