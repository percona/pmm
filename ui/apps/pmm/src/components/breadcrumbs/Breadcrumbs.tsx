import { FC } from 'react';
import {
  alpha,
  Link,
  Breadcrumbs as MuiBreadcrumbs,
  useTheme,
} from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';
import KeyboardArrowRight from '@mui/icons-material/KeyboardArrowRight';
import { Messages } from './Breadcrumbs.messages';
import { HomeLink } from 'components/home-link';

export const Breadcrumbs: FC = () => {
  const theme = useTheme();

  return (
    <MuiBreadcrumbs
      aria-label="breadcrumb"
      color="text"
      separator={<KeyboardArrowRight fontSize="small" />}
    >
      <HomeLink underline="hover" color="inherit">
        {Messages.home}
      </HomeLink>
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
