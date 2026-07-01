import GlobalStyles from '@mui/material/GlobalStyles';
import Stack from '@mui/material/Stack';
import { useHeader } from 'hooks/useHeader';
import { FC, PropsWithChildren } from 'react';

const GrafanaPageFrame: FC<PropsWithChildren> = ({ children }) => {
  const { visible: headerVisible } = useHeader();

  return (
    <>
      {headerVisible && (
        <GlobalStyles
          styles={(theme) => ({
            'html, body': {
              backgroundColor: theme.palette.background.paper,
            },
          })}
        />
      )}
      <Stack
        sx={[
          {
            flex: 1,
          },
          headerVisible && {
            p: 2,
            pt: 0,
          },
        ]}
      >
        <Stack
          sx={[
            {
              flex: 1,
            },
            headerVisible && {
              border: '1px solid',
              borderColor: 'divider',
              borderRadius: '5px',
              overflow: 'hidden',
            },
          ]}
        >
          {children}
        </Stack>
      </Stack>
    </>
  );
};

export default GrafanaPageFrame;
