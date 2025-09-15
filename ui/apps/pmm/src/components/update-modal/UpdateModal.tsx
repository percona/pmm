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

const UpdateModal: FC = () => {
  const { isLoading, status, versionInfo } = useUpdates();
  const [open, setIsOpen] = useState(true);

  if (isLoading || !versionInfo) {
    return false;
  }

  console.log(status, versionInfo);

  const handleClose = () => {
    setIsOpen(false);
  };

  return (
    <Modal
      title={Messages.title(versionInfo.latest.version)}
      open={open}
      onClose={() => setIsOpen(false)}
      disableAutoFocus
    >
      <Stack gap={1}>
        <Typography>{Messages.description}</Typography>
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
