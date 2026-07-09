import Box from '@mui/material/Box';
import { CodeBlock } from '@percona/percona-ui';
import { FC } from 'react';

export interface Props {
  query: string;
}

// Full-width container + hidden overflow so the block's frame and border
// always render inside the cell instead of getting cut off by it
const QueryCell: FC<Props> = ({ query }) => (
  <Box sx={{ width: '100%', minWidth: 0 }}>
    <CodeBlock
      content={query
        .replace(/[\n\r\t]/g, '')
        .replace(/\s{2,}/g, ' ')
        .trim()}
      language="javascript"
      sx={{ overflow: 'hidden' }}
    />
  </Box>
);

export default QueryCell;
