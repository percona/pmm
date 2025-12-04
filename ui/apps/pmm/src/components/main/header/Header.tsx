import { FC } from 'react';
import { useHeader } from 'hooks/useHeader';

const Header: FC = () => {
  const { visible, Component } = useHeader();

  if (!visible || !Component) {
    return null;
  }

  return <Component />;
};

export default Header;
