import Stack from '@mui/material/Stack';
import useHeader from 'hooks/useHeader';
import { FC, PropsWithChildren } from 'react';

const GrafanaPageFrame: FC<PropsWithChildren> = ({ children }) => {
  const { visible: headerVisible } = useHeader();

  return (
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
            borderRadius: 4,
            overflow: 'hidden',
          },
        ]}
      >
        {children}
      </Stack>
    </Stack>
  );
};

export default GrafanaPageFrame;
