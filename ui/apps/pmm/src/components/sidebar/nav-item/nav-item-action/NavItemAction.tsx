import Add from '@mui/icons-material/Add';
import IconButton from '@mui/material/IconButton';
import { SxProps, Theme } from '@mui/material/styles';
import { Tooltip } from '@percona/peak-ui';
import { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { NavItemAction as NavItemActionType } from 'types/navigation.types';

interface Props {
  action: NavItemActionType;
  testId: string;
  sx?: SxProps<Theme>;
  onClick?: () => void;
}

const NavItemAction: FC<Props> = ({
  action,
  testId,
  sx = { mr: 0.5 },
  onClick,
}) => (
  <Tooltip title={action.label} placement="right" arrow>
    <IconButton
      size="small"
      component={RouterLink}
      to={action.url}
      aria-label={action.label}
      data-testid={testId}
      sx={sx}
      onClick={onClick}
    >
      <Add fontSize="small" />
    </IconButton>
  </Tooltip>
);

export default NavItemAction;
