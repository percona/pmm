import { Autocomplete, Button, Link, TextField } from '@mui/material';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { FC } from 'react';
import { Messages } from './RealTimeSelection.messages';

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
          <Typography variant="h5">{Messages.title}</Typography>
          <Stack flexDirection="column" gap={2}>
            <Typography variant="body1" textAlign="center">
              {Messages.description}
            </Typography>
            <Autocomplete
              renderInput={(params) => (
                <TextField {...params} label={Messages.selectCluster} />
              )}
              options={[]}
            />
            <Autocomplete
              renderInput={(params) => (
                <TextField {...params} label={Messages.selectService} />
              )}
              options={[]}
            />
            <Button disabled variant="contained" color="primary">
              {Messages.start}
            </Button>
            <Stack direction="column" gap={1} mt={1}>
              <Typography variant="body1" textAlign="center">
                {Messages.featureAvailable}
              </Typography>
              <Stack justifyContent="center" flexDirection="row" gap={2}>
                <Link href="#" rel="noopener noreferrer" target="_blank">
                  {Messages.documentation}
                </Link>
                <Link href="#" rel="noopener noreferrer" target="_blank">
                  {Messages.provideFeedback}
                </Link>
              </Stack>
            </Stack>
          </Stack>
        </Stack>
      </Stack>
    </Page>
  );
};

export default RealTimeSelectionPage;
