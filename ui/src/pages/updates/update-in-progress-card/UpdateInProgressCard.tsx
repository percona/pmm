import { FC } from 'react';
import {
  Card,
  CardContent,
  Stack,
  Typography,
  Chip,
  Link,
} from '@mui/material';
import { UpdateInfo } from '../update-info';
import { UpdateInProgressCardProps } from './UpdateInProgressCard.types';
import { Messages } from './UpdateInProgressCard.messages';
import { UpdateProgress } from './update-progress/UpdateProgress';
import { UpdateStatus } from 'types/updates.types';
import { PMM_HOME_URL } from 'constants';

export const UpdateInProgressCard: FC<UpdateInProgressCardProps> = ({
  versionInfo,
  status,
}) => (
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
            <Chip label="In progress" color="warning" />
          </Stack>
          <UpdateInfo />
        </Stack>
        <UpdateProgress status={status} />
        {status === UpdateStatus.Completed && (
          <Stack direction="row" justifyContent="flex-end">
            <Link href={PMM_HOME_URL} underline="none">
              {Messages.home}
            </Link>
          </Stack>
        )}
      </Stack>
    </CardContent>
  </Card>
);
