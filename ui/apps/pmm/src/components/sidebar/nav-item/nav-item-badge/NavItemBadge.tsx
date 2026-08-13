import Chip from '@mui/material/Chip';
import type { FC} from 'react';
import { isValidElement } from 'react';
import type { NavItem } from 'types/navigation.types';

interface Props {
  badge: NavItem['badge'];
}

const NavItemBadge: FC<Props> = ({ badge: Badge }) => {
  if (isValidElement(Badge)) {
    return Badge;
  }

  if (typeof Badge === 'object' && Badge !== null) {
    return <Chip size="small" color="warning" variant="outlined" {...Badge} />;
  }

  return null;
};

export default NavItemBadge;
