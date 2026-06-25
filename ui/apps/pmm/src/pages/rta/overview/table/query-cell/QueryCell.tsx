import { CodeBlock } from 'components/code-block';
import Tooltip from '@mui/material/Tooltip';
import { FC } from 'react';
import { Messages } from './QueryCell.messages';

export interface Props {
  query: string;
  truncated?: boolean;
  language?: 'mongodb' | 'sql';
}

const QueryCell: FC<Props> = ({ query, truncated, language = 'mongodb' }) => {
  const normalized = query
    .replace(/[\n\r\t]/g, '')
    .replace(/\s{2,}/g, ' ')
    .trim();

  const content = (
    <CodeBlock
      code={normalized}
      containerProps={{
        sx: {
          width: '100%',
          maxHeight: '50px',
        },
      }}
      language={language}
    />
  );

  if (!truncated) {
    return content;
  }

  return (
    <Tooltip title={Messages.truncatedTooltip} arrow>
      <span>{content}</span>
    </Tooltip>
  );
};

export default QueryCell;
