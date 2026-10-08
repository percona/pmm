import Add from '@mui/icons-material/Add';
import IconButton from '@mui/material/IconButton';
import { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { NavItemAction as NavItemActionType } from 'types/navigation.types';

interface Props {
  action: NavItemActionType;
  testId: string;
}

const NavItemAction: FC<Props> = ({ action, testId }) => {
  const Icon = action.icon ?? Add;

  return (
    <IconButton
      size="small"
      component={RouterLink}
      to={action.url}
      aria-label={action.label}
      title={action.label}
      data-testid={testId}
      sx={{ mr: 1 }}
    >
      <Icon fontSize="small" />
    </IconButton>
  );
};

export default NavItemAction;
