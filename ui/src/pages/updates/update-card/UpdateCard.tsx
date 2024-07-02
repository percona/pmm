import {
  Button,
  Card,
  CardActions,
  CardContent,
  Stack,
  Link,
  Typography,
  Skeleton,
  Alert,
} from '@mui/material';
import { FC, useMemo } from 'react';
import { formatTimestamp } from 'utils/formatTimestamp';
import { PMM_HOME_URL } from 'constants';
import { Messages } from './UpdateCard.messages';
import { FetchingIcon } from 'components/fetching-icon';
import { useCheckUpdates } from 'hooks/api/useUpdates';
import { formatVersion } from './UpdateCard.utils';

export const UpdateCard: FC = () => {
  const { isLoading, data, error, isRefetching, refetch } = useCheckUpdates();
  const isUpToDate = useMemo(
    () => data?.installed.version === data?.latest?.version,
    [data]
  );

  if (isLoading)
    return (
      <Card>
        <CardContent>
          <Stack spacing={1}>
            <Skeleton />
            <Skeleton />
          </Stack>
        </CardContent>
      </Card>
    );

  if (!data || error) {
    return (
      <Card>
        <CardContent>
          <Alert severity="error">{Messages.fetchError}</Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card sx={{ p: 1 }}>
      <CardContent>
        {isUpToDate && (
          <Alert
            severity="success"
            sx={{
              mb: 2,
            }}
          >
            {Messages.upToDate}
          </Alert>
        )}
        <Stack spacing={1}>
          <Typography variant="body1">
            <Typography fontWeight="bold" component="strong">
              {Messages.runningVersion}
            </Typography>
            {data?.installed && formatVersion(data.installed)}
          </Typography>
          {data.lastCheck && (
            <Typography variant="body1">
              <Typography fontWeight="bold" component="strong">
                {Messages.lastChecked}
              </Typography>{' '}
              {formatTimestamp(data?.lastCheck)}
            </Typography>
          )}
        </Stack>
      </CardContent>
      <CardActions>
        <Button
          startIcon={<FetchingIcon isFetching={isRefetching} />}
          variant="contained"
          onClick={() => refetch()}
        >
          {isRefetching ? Messages.checking : Messages.checkNow}
        </Button>
        <Link href={PMM_HOME_URL}>
          <Button variant="outlined">{Messages.home}</Button>
        </Link>
      </CardActions>
    </Card>
  );
};
