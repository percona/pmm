import { CodeBlock } from 'components/code-block';
import { FC } from 'react';
import { CodeLanguage } from 'types/util.types';

export interface Props {
  query: string;
  language?: CodeLanguage;
}

const QueryCell: FC<Props> = ({ query, language = 'mongodb' }) => (
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
    language={language}
  />
);

export default QueryCell;
