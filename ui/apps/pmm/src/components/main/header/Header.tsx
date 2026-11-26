import { FC } from 'react';
import { useHeader } from 'hooks/useHeader';
import Stack from '@mui/material/Stack';

const Header: FC = () => {
  const { visible, Component } = useHeader();

  if (!visible || !Component) {
    return null;
  }

  return (
    <Stack sx={{ justifyContent: 'center' }}>
      <Component />
    </Stack>
  );
};

export default Header;
