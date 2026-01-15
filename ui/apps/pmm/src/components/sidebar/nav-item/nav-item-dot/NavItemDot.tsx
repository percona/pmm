import Badge from '@mui/material/Badge';
import { FC, memo, PropsWithChildren } from 'react';

interface Props extends PropsWithChildren {
  show: boolean;
}

const NavItemDot: FC<Props> = memo(({ show, children }) =>
  show ? (
    <Badge variant="dot" color="warning" data-testid="navitem-dot">
      {children}
    </Badge>
  ) : (
    <>{children}</>
  )
);

export default NavItemDot;
