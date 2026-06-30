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
import { FC, useState } from 'react';
import { PMM_HOME_URL } from 'lib/constants';
import { Messages } from './UpdateCard.messages';
import { FetchingIcon } from 'components/fetching-icon';
import { useCheckUpdates, useStartUpdate } from 'hooks/api/useUpdates';
import { formatVersion } from './UpdateCard.utils';
import { enqueueSnackbar } from 'notistack';
import { UpdateStatus } from 'types/updates.types';
import KeyboardDoubleArrowUp from '@mui/icons-material/KeyboardDoubleArrowUp';
import { UpdateInfo } from '../update-info';
import { UpdateInProgressCard } from '../update-in-progress-card';
import { useUpdates } from 'contexts/updates';
import { ChangeLog } from '../change-log';
import { capitalize } from 'utils/text.utils';
import { DEPRECATION_DOCKER_UPGRADE_HREF, DEPRECATION_HELM_UPGRADE_HREF, DEPRECATION_PODMAN_UPGRADE_HREF } from './UpdateCard.constants';

export const UpdateCard: FC = () => {
  const { inProgress, status, setStatus } = useUpdates();
  const { isLoading, data, error, isRefetching, refetch } = useCheckUpdates();
  const { mutate: startUpdate } = useStartUpdate();
  const [authToken, setAuthToken] = useState<string>();

  const handleStartUpdate = async () => {
    setStatus(UpdateStatus.Updating);
    startUpdate(
      {},
      {
        onSuccess: async (response) => {
          if (response) {
            setStatus(UpdateStatus.Restarting);
            setAuthToken(response.authToken);
          }
        },
        onError: (e) => {
          const message = e.isAxiosError ? e.response?.data.message : e.message;
          setStatus(UpdateStatus.Error);
          enqueueSnackbar(message ? capitalize(message) : Messages.error, {
            variant: 'error',
          });
        },
      }
    );
  };

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

  if (inProgress && data.latest) {
    return (
      <UpdateInProgressCard
        versionInfo={data.latest}
        status={status}
        authToken={authToken}
      />
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
        <>
          {/* TODO temporary solution for link color */}
          <Alert severity="warning" sx={{ mb: 2, '& a': { color: 'inherit', textDecorationColor: 'inherit' } }}>
            <Typography variant="body1">
              <strong>{Messages.deprecation.heading}</strong>
              {Messages.deprecation.paragraph1BeforeUpdateNow}
              <strong>{Messages.updateNow}</strong>
              {Messages.deprecation.paragraph1AfterUpdateNow}
            </Typography>
            <Typography>
              {Messages.deprecation.viaIntro}
              <Link href={DEPRECATION_DOCKER_UPGRADE_HREF} target="_blank" rel="noopener noreferrer">
                {Messages.deprecation.docker}
              </Link>
              {Messages.deprecation.afterDocker}
              <Link href={DEPRECATION_PODMAN_UPGRADE_HREF} target="_blank" rel="noopener noreferrer">
                {Messages.deprecation.podman}
              </Link>
              {Messages.deprecation.afterPodman}
              <Link href={DEPRECATION_HELM_UPGRADE_HREF} target="_blank" rel="noopener noreferrer">
                {Messages.deprecation.helm}
              </Link>
              {Messages.deprecation.afterHelm}
            </Typography>
            <Typography>{Messages.deprecation.reminder}</Typography>
          </Alert>
          <CardActions>
            <Button
              endIcon={<KeyboardDoubleArrowUp />}
              variant="contained"
              onClick={handleStartUpdate}
            >
              {Messages.updateNow}
            </Button>
          </CardActions>
        </>
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
