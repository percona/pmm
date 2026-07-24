import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Card } from '@percona/percona-ui';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';

interface Props {
  pageTitle?: string;
  feature: string;
}

const FeatureCheck: FC<Props> = ({ feature, pageTitle }) => {
  const { user } = useUser();
  return (
    <Page title={pageTitle}>
      <Card
        dataTestId="empty-block"
        content={
          <Stack flex={1} py={4} alignItems="center" justifyContent="center">
            <Typography>
              {`${feature} is disabled. `}
              {user?.isPMMAdmin ? (
                <>
                  {'You can enable it in '}
                  <Link
                    data-testid="settings-link"
                    component={RouterLink}
                    to="/settings/advanced-settings"
                  >
                    PMM Settings
                  </Link>
                  .
                </>
              ) : (
                'Ask your admin to enable it in PMM Settings.'
              )}
            </Typography>
          </Stack>
        }
        sx={{ width: '100%', mt: 2 }}
      />
    </Page>
  );
};

export default FeatureCheck;
