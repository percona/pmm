import GlobalStyles from '@mui/material/GlobalStyles';
import Stack, { StackProps } from '@mui/material/Stack';
import { HEADER_HEIGHT } from 'components/main/header/Header.constants';
import { FC } from 'react';

const RealtimePage: FC<StackProps> = ({ children }) => (
  <>
    <GlobalStyles
      styles={(theme) => ({
        'html, body': {
          backgroundColor: theme.palette.background.paper,
        },
      })}
    />
    <Stack
      direction="column"
      gap={2}
      p={2}
      sx={{
        flex: 1,
        height: '100%',
        minHeight: 0,
        position: 'relative',
        maxHeight: `calc(100vh - ${HEADER_HEIGHT}px)`, // Account for header height
        // There should be a better way to avoid using maxHeight, or forcing hidden overflow. To be improved.
        overflow: 'hidden',
        display: 'flex',
      }}
    >
      {children}
    </Stack>
  </>
);

export default RealtimePage;
