import { Button, Stack, Typography } from '@mui/material';
import ArrowBack from '@mui/icons-material/ArrowBack';
import { NothingFoundIllustration } from '@percona/peak-ui';
import { Page } from 'components/page';
import { PMM_NEW_NAV_HOME_URL } from 'lib/constants';
import type { FC } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { Messages } from './NotFoundPage.messages';

export const NotFoundPage: FC = () => {
  const navigate = useNavigate();
  // React Router stores the position of the current entry in history state.
  // Index 0 means the user landed here directly (bookmark, external link, or
  // a redirect that replaced the first entry), so there is nothing to go back to.
  const canGoBack = (window.history.state?.idx ?? 0) > 0;

  return (
    <Page title={Messages.title} hideTitle surface="paper" footer={null}>
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
            aria-hidden
            sx={{ height: 160, width: 160, mb: -3 }}
          />
          <Stack gap={1}>
            <Typography variant="h3" component="h1">
              {Messages.title}
            </Typography>
            <Typography variant="body1" color="text.secondary">
              {Messages.description}
            </Typography>
          </Stack>
          <Stack
            direction={{ xs: 'column', sm: 'row-reverse' }}
            gap={1}
            sx={{ width: { xs: '100%', sm: 'auto' } }}
          >
            <Button
              variant="contained"
              size="large"
              component={RouterLink}
              to={PMM_NEW_NAV_HOME_URL}
              data-testid="not-found-home-button"
            >
              {Messages.goHome}
            </Button>
            {canGoBack && (
              <Button
                variant="outlined"
                size="large"
                startIcon={<ArrowBack />}
                onClick={() => navigate(-1)}
                data-testid="not-found-back-button"
              >
                {Messages.goBack}
              </Button>
            )}
          </Stack>
        </Stack>
      </Stack>
    </Page>
  );
};
