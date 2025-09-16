import { Modal } from 'components/modal';
import { useUpdates } from 'contexts/updates';
import { FC, useState } from 'react';
import { Messages } from './UpdateModal.messages';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import Link from '@mui/material/Link';
import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import { PMM_NEW_NAV_UPDATES_PATH } from 'lib/constants';
import Snackbar from '@mui/material/Snackbar';
import CloseIcon from '@mui/icons-material/Close';
import Card from '@mui/material/Card';
import IconButton from '@mui/material/IconButton';

const UpdateModal: FC = () => {
  const { isLoading, versionInfo } = useUpdates();
  const [open, setIsOpen] = useState(true);
  const [isFirstAttempt, setIsFirstAttempt] = useState(true);

  if (isLoading || !versionInfo) {
    return false;
  }

  const handleClose = () => {
    setIsOpen(false);

    if (isFirstAttempt) {
      setIsFirstAttempt(false);

      setTimeout(() => {
        setIsOpen(true);
      }, 2500);
    }
  };

  if (!isFirstAttempt) {
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
        <Box component="ul" sx={{ my: 1 }}>
          <li>
            {Messages.more}
            <Link
              href={versionInfo.latest.releaseNotesUrl}
              target="_blank"
              rel="noopener noreferrer"
            >
              {Messages.releaseNotes}
            </Link>
          </li>
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
