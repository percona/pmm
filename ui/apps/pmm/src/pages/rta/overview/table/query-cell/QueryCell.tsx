import Box from '@mui/material/Box';
import { CodeBlock } from '@percona/percona-ui';
import { FC } from 'react';

export interface Props {
  query: string;
}

// Fade the clipped text out toward the right edge as a cut-off affordance.
// Masking the content (not overlaying a gradient) keeps it independent of the
// prism scheme's background color in either mode.
const fadeMask =
  'linear-gradient(to right, black calc(100% - 48px), transparent)';

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
      sx={{
        overflow: 'hidden',
        '& > code': {
          display: 'block',
          maskImage: fadeMask,
          maskRepeat: 'no-repeat',
          maskSize: '100% 100%',
          WebkitMaskImage: fadeMask,
          WebkitMaskRepeat: 'no-repeat',
          WebkitMaskSize: '100% 100%',
        },
      }}
    />
  </Box>
);

export default QueryCell;
