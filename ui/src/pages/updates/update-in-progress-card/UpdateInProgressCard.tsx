import { FC } from 'react';
import {
  Card,
  CardContent,
  Stack,
  Typography,
  Chip,
  Button,
} from '@mui/material';
import { UpdateInfo } from '../update-info';
import { UpdateInProgressCardProps } from './UpdateInProgressCard.types';
import { Messages } from './UpdateInProgressCard.messages';
import { UpdateProgress } from './update-progress/UpdateProgress';
import { UpdateStatus } from 'types/updates.types';
import { UpdateLog } from '../update-log';
import { useNavigate } from 'react-router-dom';

export const UpdateInProgressCard: FC<UpdateInProgressCardProps> = ({
  versionInfo,
  authToken,
  status,
}) => {
  const navigate = useNavigate();

  return (
    <Card>
      <CardContent>
        <Stack spacing={2}>
          <Stack>
            <Stack
              direction="row"
              spacing={1}
              sx={{
                alignItems: 'center',
              }}
            >
              <Typography variant="h5">
                {versionInfo.version && Messages.title(versionInfo.version)}
              </Typography>
              <Chip label={Messages.inProgress} color="warning" />
            </Stack>
            <UpdateInfo />
          </Stack>
          <UpdateProgress status={status} />
          <Stack
            direction="row"
            justifyContent={
              status === UpdateStatus.Completed ? 'space-between' : 'flex-end'
            }
          >
            {status === UpdateStatus.Completed && (
              <Button
                variant="contained"
                onClick={() => navigate('/updates/clients')}
              >
                {Messages.next}
              </Button>
            )}
            {!!authToken && <UpdateLog authToken={authToken} />}
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
};
