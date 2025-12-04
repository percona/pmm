import { Autocomplete, Button, Link, TextField } from '@mui/material';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { FC } from 'react';

const RealTimeSelectionPage: FC = () => {
  return (
    <Page footer={null}>
      <Stack flex={1} alignItems="center" justifyContent="center">
        <Stack
          alignItems="center"
          justifyContent="center"
          gap={1}
          flex={1}
          sx={{ width: 320 }}
        >
          <Typography variant="h5">Real-Time Query Analysis</Typography>
          <Stack flexDirection="column" gap={2}>
            <Typography variant="body1" textAlign="center">
              Select a service to start a new real-time session, showing all
              executing queries and performance metrics.
            </Typography>
            <Autocomplete
              renderInput={(params) => (
                <TextField {...params} label="Select a cluster" />
              )}
              options={[]}
            />
            <Autocomplete
              renderInput={(params) => (
                <TextField {...params} label="Select a service" />
              )}
              options={[]}
            />
            <Button variant="contained" color="primary">
              Start
            </Button>
            <Typography variant="body1" textAlign="center">
              Feature available for MongoDB database technology only, more to
              come soon.
            </Typography>
            <Stack justifyContent="center" flexDirection="row" gap={2}>
              <Link>Documentation</Link>
              <Link>Provide Feedback</Link>
            </Stack>
          </Stack>
        </Stack>
      </Stack>
    </Page>
  );
};

export default RealTimeSelectionPage;
