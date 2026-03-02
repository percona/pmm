import { FC } from 'react';
import { useHeader } from 'hooks/useHeader';
import Stack from '@mui/material/Stack';
import { HEADER_HEIGHT } from './Header.constants';

const Header: FC = () => {
  const { visible, Component } = useHeader();

  if (!visible || !Component) {
    return null;
  }

  return (
    <Stack sx={{ height: HEADER_HEIGHT, justifyContent: 'center' }}>
      <Component />
    </Stack>
  );
};

export default Header;
