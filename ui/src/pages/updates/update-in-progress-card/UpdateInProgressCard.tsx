import { FC } from 'react';
import {
  Card,
  CardContent,
  Stack,
  Typography,
  LinearProgress,
  Chip,
} from '@mui/material';
import { UpdateInfo } from '../update-info';
import { UpdateInProgressCardProps } from './UpdateInProgressCard.types';
import { Messages } from './UpdateInProgressCard.messages';

export const UpdateInProgressCard: FC<UpdateInProgressCardProps> = ({
  versionInfo,
}) => (
  <Card>
    <CardContent>
      <Stack spacing={1}>
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
        <Stack
          sx={{
            pt: 2,
          }}
        >
          <Typography
            sx={{
              fontSize: 12,
              pb: 1,
            }}
          >
            {Messages.updating}
          </Typography>
          <LinearProgress />
        </Stack>
      </Stack>
    </CardContent>
  </Card>
);
