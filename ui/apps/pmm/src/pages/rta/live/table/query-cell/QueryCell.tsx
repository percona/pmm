import { CodeBlock } from 'components/code-block';
import { FC } from 'react';
import { QueryCellProps } from './QueryCell.types';

const QueryCell: FC<QueryCellProps> = ({ query }) => (
  <CodeBlock
    code={query.replace(/\n/g, '').replace(/  /g, '')}
    language="mongodb"
  />
);

export default QueryCell;
