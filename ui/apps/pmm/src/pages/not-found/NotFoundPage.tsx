import { Button, Link, Stack, Typography } from '@mui/material';
import ArrowBack from '@mui/icons-material/ArrowBack';
import { CodeBlock, NothingFoundIllustration } from '@percona/peak-ui';
import { Page } from 'components/page';
import { PMM_BASE_PATH, PMM_NEW_NAV_HOME_URL } from 'lib/constants';
import type { FC } from 'react';
import { Link as RouterLink, useLocation, useNavigate } from 'react-router-dom';
import { Messages } from './NotFoundPage.messages';
import { QUICK_LINKS } from './NotFoundPage.constants';

export const NotFoundPage: FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const requestedPath = `${PMM_BASE_PATH}${location.pathname}${location.search}`;
  // React Router assigns the key 'default' to the first entry of a session,
  // i.e. when the user landed here directly from a bookmark or external link.
  const canGoBack = location.key !== 'default';

  return (
    <Page title={Messages.title}>
      <Stack
        sx={{
          flex: 1,
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Stack
          gap={3}
          sx={{
            p: 2,
            width: '100%',
            maxWidth: 480,
            alignItems: 'center',
            textAlign: 'center',
          }}
          data-testid="not-found-page"
        >
          <NothingFoundIllustration
            color="primary"
            sx={{ height: 192, width: 192, mb: -3 }}
          />
          <Stack gap={1}>
            <Typography variant="h6">{Messages.heading}</Typography>
            <Typography variant="body1" color="text.secondary">
              {Messages.description}
            </Typography>
          </Stack>
          <Stack gap={0.5} sx={{ width: '100%', textAlign: 'left' }}>
            <Typography variant="overline" color="text.secondary">
              {Messages.requestedPath}
            </Typography>
            <CodeBlock
              content={requestedPath}
              wrap
              data-testid="not-found-requested-path"
            />
          </Stack>
          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            gap={1}
            sx={{ width: { xs: '100%', sm: 'auto' } }}
          >
            <Button
              variant="contained"
              component={RouterLink}
              to={PMM_NEW_NAV_HOME_URL}
              data-testid="not-found-home-button"
            >
              {Messages.goHome}
            </Button>
            {canGoBack && (
              <Button
                variant="outlined"
                startIcon={<ArrowBack />}
                onClick={() => navigate(-1)}
                data-testid="not-found-back-button"
              >
                {Messages.goBack}
              </Button>
            )}
          </Stack>
          <Stack
            direction="row"
            gap={1}
            sx={{ flexWrap: 'wrap', justifyContent: 'center' }}
          >
            <Typography variant="body2" color="text.secondary">
              {Messages.quickLinks}
            </Typography>
            {QUICK_LINKS.map((link) => (
              <Link
                key={link.id}
                component={RouterLink}
                to={link.to}
                variant="body2"
                data-testid={`not-found-link-${link.id}`}
              >
                {link.label}
              </Link>
            ))}
          </Stack>
        </Stack>
      </Stack>
    </Page>
  );
};
