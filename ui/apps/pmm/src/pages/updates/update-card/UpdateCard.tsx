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
import { FC } from 'react';
import { PMM_HOME_URL } from 'lib/constants';
import { Messages } from './UpdateCard.messages';
import { FetchingIcon } from 'components/fetching-icon';
import { useCheckUpdates } from 'hooks/api/useUpdates';
import { formatVersion } from './UpdateCard.utils';
import { UpdateStatus } from 'types/updates.types';
import CallMadeIcon from '@mui/icons-material/CallMade';
import { UpdateInfo } from '../update-info';
import { useUpdates } from 'contexts/updates';
import { ChangeLog } from '../change-log';
import { UPGRADE_DOCS_HREF } from './UpdateCard.constants';

export const UpdateCard: FC = () => {
  const { status } = useUpdates();
  const { isLoading, data, error, isRefetching, refetch } = useCheckUpdates();

  if (isLoading)
    return (
      <Card variant="outlined">
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
      <Card variant="outlined">
        <CardContent>
          <Alert severity="error">{Messages.fetchError}</Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card variant="outlined" sx={{ p: 1 }}>
      <CardContent>
        {status === UpdateStatus.UpToDate && (
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
          {data.updateAvailable && data?.latest?.version && (
            <Typography variant="h4">
              {Messages.newUpdateAvailable(data.latest.version)}
            </Typography>
          )}
          <Typography>
            <Typography fontWeight="bold" component="strong">
              {Messages.runningVersion}
            </Typography>
            {data?.installed && formatVersion(data.installed)}
          </Typography>
          {data.updateAvailable && data.latest && (
            <Typography>
              <Typography fontWeight="bold" component="strong">
                {Messages.newVersion}
              </Typography>
              {data?.latest && formatVersion(data.latest)}
            </Typography>
          )}
        </Stack>
        {data.updateAvailable && <UpdateInfo />}
      </CardContent>
      {data.updateAvailable ? (
          <CardActions>
            <Button
              endIcon={<CallMadeIcon />}
              variant="contained"
              href={UPGRADE_DOCS_HREF}
              target="_blank"
              rel="noopener noreferrer"
            >
              {Messages.howToUpdateDocs}
            </Button>
          </CardActions>
      ) : (
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
      )}
      <ChangeLog />
    </Card>
  );
};
