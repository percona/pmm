import Badge from '@mui/material/Badge';
import { FC, PropsWithChildren } from 'react';

interface Props extends PropsWithChildren {
  show: boolean;
}

const NavItemDot: FC<Props> = ({ show, children }) =>
  show ? (
    <Badge variant="dot" color="warning">
      {children}
    </Badge>
  ) : (
    <>{children}</>
  );

export default NavItemDot;
