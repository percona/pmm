import type { FC } from 'react';
import { AppHeader } from 'components/app-header';
import { useHeader } from 'hooks/useHeader';

const Header: FC = () => {
  const { page, inset } = useHeader();

  if (!page) {
    return null;
  }

  return (
    <AppHeader
      title={page.app.text ?? ''}
      starId={page.starrable ? page.app.id : undefined}
      inset={inset}
    />
  );
};

export default Header;
