import { Modal } from 'components/modal';
import { useUpdates } from 'contexts/updates';
import { FC, useEffect, useState } from 'react';
import { Messages } from './UpdateModal.messages';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import Link from '@mui/material/Link';
import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import {
  PMM_NEW_NAV_UPDATES_PATH,
  SHOW_UPDATE_INFO_DELAY_MS,
} from 'lib/constants';
import Snackbar from '@mui/material/Snackbar';
import CloseIcon from '@mui/icons-material/Close';
import Card from '@mui/material/Card';
import IconButton from '@mui/material/IconButton';
import { parseReleaseHighlights } from './UpdateModal.utils';
import { ReleaseNotes } from 'pages/updates/change-log/release-notes';
import { useSnooze } from 'hooks/snooze';

const UpdateModal: FC = () => {
  const { isLoading, versionInfo } = useUpdates();
  const [open, setIsOpen] = useState(false);
  const highlights = parseReleaseHighlights(
    versionInfo?.latest.releaseNotesText
  );
  const { snoozeUpdate, snoozeCount } = useSnooze();

  const handleClose = () => {
    setIsOpen(false);

    snoozeUpdate();
  };

  useEffect(() => {
    setTimeout(() => {
      setIsOpen(true);
    }, SHOW_UPDATE_INFO_DELAY_MS);
  }, []);

  if (isLoading || !versionInfo) {
    return false;
  }

  if (snoozeCount > 0) {
    return (
      <Snackbar
        open={open}
        anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
        onClose={handleClose}
      >
        <Card
          elevation={12}
          sx={{
            width: 500,
            p: 2,
          }}
        >
          <Stack gap={2}>
            <Stack direction="row">
              <Typography>
                <Typography fontWeight="bold" display="inline-block">
                  {Messages.title(versionInfo.latest.version)}
                </Typography>
                {Messages.descriptionSnack}
              </Typography>
              <IconButton
                onClick={handleClose}
                sx={{
                  alignSelf: 'flex-start',
                }}
              >
                <CloseIcon sx={{ width: 20, height: 20 }} />
              </IconButton>
            </Stack>
            <Stack gap={1} direction="row">
              <Button
                variant="contained"
                onClick={handleClose}
                component={RouterLink}
                to={PMM_NEW_NAV_UPDATES_PATH}
              >
                {Messages.goToUpdates}
              </Button>
              <Button onClick={handleClose}>{Messages.remindMe}</Button>
            </Stack>
          </Stack>
        </Card>
      </Snackbar>
    );
  }

  return (
    <Modal
      title={Messages.title(versionInfo.latest.version)}
      open={open}
      onClose={handleClose}
      disableAutoFocus
    >
      <Stack gap={1}>
        <Typography>{Messages.descriptionModal}</Typography>
        <Typography variant="h6">{Messages.highlights}</Typography>
        <Box sx={{ my: 1 }}>
          <Box
            sx={{
              mb: 1,

              '& p': {
                m: 0,
              },
            }}
          >
            <ReleaseNotes content={highlights} />
          </Box>
          {Messages.more}
          <Link
            href={versionInfo.latest.releaseNotesUrl}
            target="_blank"
            rel="noopener noreferrer"
          >
            {Messages.releaseNotes}
          </Link>
        </Box>
        <Stack direction="row" justifyContent="end" sx={{ gap: 1, pt: 2 }}>
          <Button variant="text" onClick={handleClose}>
            {Messages.remindMe}
          </Button>
          <Button
            variant="contained"
            onClick={handleClose}
            component={RouterLink}
            to={PMM_NEW_NAV_UPDATES_PATH}
          >
            {Messages.goToUpdates}
          </Button>
        </Stack>
      </Stack>
    </Modal>
  );
};

export default UpdateModal;
