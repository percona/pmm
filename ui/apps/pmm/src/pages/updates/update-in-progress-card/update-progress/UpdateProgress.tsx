import { FC } from 'react';
import { UpdateProgressProps } from './UpdateProgress.types';
import {
  LinearProgress,
  Stack,
  Typography,
  linearProgressClasses,
} from '@mui/material';
import { UpdateStatus } from 'types/updates.types';
import { Messages } from './UpdateProgress.messages';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';

export const UpdateProgress: FC<UpdateProgressProps> = ({ status }) => {
  const getStatusPercentage = (status: UpdateStatus) => {
    switch (status) {
      case UpdateStatus.Updating:
        return 33;
      case UpdateStatus.Restarting:
        return 66;
      case UpdateStatus.Completed:
        return 100;
      default:
        return 0;
    }
  };

  const getStatusMessage = (status: UpdateStatus) => {
    switch (status) {
      case UpdateStatus.Updating:
        return Messages.updating;
      case UpdateStatus.Restarting:
        return Messages.restarting;
      case UpdateStatus.Completed:
        return Messages.completed;
      default:
        return '';
    }
  };

  return (
    <Stack>
      <Stack
        direction="row"
        alignItems="center"
        sx={{
          gap: 0.5,
          pb: 1,
        }}
      >
        {status === UpdateStatus.Completed && (
          <CheckCircleIcon
            fontSize="small"
            sx={{
              color: '#008C71',
            }}
          />
        )}
        <Typography
          sx={{
            fontSize: 12,
          }}
        >
          {getStatusMessage(status)}
        </Typography>
      </Stack>
      <LinearProgress
        variant="determinate"
        value={getStatusPercentage(status)}
        sx={() => ({
          [`& .${linearProgressClasses.bar}`]: {
            backgroundColor:
              status === UpdateStatus.Completed ? '#008C71' : '#606C86',
          },
        })}
      />
    </Stack>
  );
};
