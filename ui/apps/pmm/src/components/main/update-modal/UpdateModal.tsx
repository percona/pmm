import { Modal } from 'components/modal';
import { useUpdates } from 'contexts/updates';
import { FC, useEffect, useState } from 'react';
import { Messages } from './UpdateModal.messages';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import Link from '@mui/material/Link';
import { Link as RouterLink, useLocation } from 'react-router-dom';
import Button from '@mui/material/Button';
import { PMM_NEW_NAV_UPDATES_PATH } from 'lib/constants';
import Snackbar from '@mui/material/Snackbar';
import CloseIcon from '@mui/icons-material/Close';
import Card from '@mui/material/Card';
import IconButton from '@mui/material/IconButton';
import { useSnooze } from 'hooks/updates';

const UpdateModal: FC = () => {
  const { isLoading, versionInfo } = useUpdates();
  const [open, setIsOpen] = useState(false);
  const { snoozeUpdate, snoozeActive, snoozeCount } = useSnooze();
  const location = useLocation();
  const isOnUpdatesPage = location.pathname.startsWith(
    PMM_NEW_NAV_UPDATES_PATH
  );
  const latestVersion = versionInfo?.latest.version || '';
  const releaseNotesUrl = versionInfo?.latest?.releaseNotesUrl ?? '';
  const updateAvailable = Boolean(versionInfo?.updateAvailable);

  const handleClose = () => {
    setIsOpen(false);

    snoozeUpdate();
  };

  useEffect(() => {
    if (updateAvailable && !snoozeActive) {
      setIsOpen(true);
    }
  }, [updateAvailable, snoozeActive]);

  // don't show if already on updates page
  if (isLoading || !open || isOnUpdatesPage) {
    return null;
  }

  if (snoozeCount > 1) {
    return (
      <Snackbar
        open={open}
        anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
        onClose={handleClose}
        ClickAwayListenerProps={{
          mouseEvent: false,
        }}
      >
        <Card
          elevation={12}
          sx={{
            width: 500,
            p: 2,
          }}
          data-testid="update-modal-snackbar"
        >
          <Stack gap={2}>
            <Stack direction="row">
              <Typography>
                <Typography
                  component="span"
                  fontWeight="bold"
                  display="inline-block"
                  data-testid="update-modal-title"
                >
                  {Messages.title(latestVersion)}
                </Typography>
                <span data-testid="update-modal-snackbar-description">
                  {Messages.descriptionSnack}
                </span>
              </Typography>
              <IconButton
                data-testid="update-modal-close-button"
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
                data-testid="update-modal-go-to-updates-button"
              >
                {Messages.goToUpdates}
              </Button>
              <Button
                data-testid="update-modal-remind-me-button"
                onClick={handleClose}
              >
                {Messages.remindMe}
              </Button>
            </Stack>
          </Stack>
        </Card>
      </Snackbar>
    );
  }

  return (
    <Modal
      title={Messages.title(latestVersion)}
      open={open}
      onClose={handleClose}
      disableAutoFocus
    >
      <Stack gap={1}>
        <Typography data-testid="update-modal-description">
          {Messages.descriptionModal}
        </Typography>
        {!!releaseNotesUrl && (
          <Typography
            data-testid="update-modal-description-release-notes"
            sx={{ py: 2 }}
          >
            {Messages.check}
            <Link
              href={releaseNotesUrl}
              target="_blank"
              rel="noopener noreferrer"
              data-testid="update-modal-release-notes-link"
            >
              {Messages.releaseNotes}
            </Link>
            {Messages.toSee}
          </Typography>
        )}
        <Stack
          direction="row"
          justifyContent="end"
          sx={{ gap: 1, pt: 2, alignSelf: 'flex-end' }}
        >
          <Button
            data-testid="update-modal-remind-me-button"
            variant="text"
            onClick={handleClose}
          >
            {Messages.remindMe}
          </Button>
          <Button
            data-testid="update-modal-go-to-updates-button"
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
