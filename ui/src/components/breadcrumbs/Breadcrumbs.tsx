import { FC } from 'react';
import {
  alpha,
  Link,
  Breadcrumbs as MuiBreadcrumbs,
  useTheme,
} from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';
import { KeyboardArrowRight } from '@mui/icons-material';
import { PMM_HOME_URL } from 'constants';
import { Messages } from './Breadcrumbs.messages';

export const Breadcrumbs: FC = () => {
  const theme = useTheme();

  return (
    <MuiBreadcrumbs
      aria-label="breadcrumb"
      color="text"
      separator={<KeyboardArrowRight fontSize="small" />}
    >
      <Link underline="hover" color="inherit" href={PMM_HOME_URL}>
        {Messages.home}
      </Link>
      <Link
        underline="hover"
        component={RouterLink}
        color={alpha(theme.typography.body1.color || '#fff', 0.75)}
        to="/updates"
      >
        {Messages.updates}
      </Link>
    </MuiBreadcrumbs>
  );
};
