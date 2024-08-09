import { Modal } from 'components/modal';
import { FC } from 'react';
import { ClientsModalProps } from './ClientsModal.types';
import { Button, Link, Stack, Typography } from '@mui/material';
import { Messages } from './ClientsModal.messages';
import { PMM_HOME_URL } from 'constants';

export const ClientsModal: FC<ClientsModalProps> = ({ isOpen, onClose }) => (
  <Modal
    title={Messages.title}
    open={isOpen}
    onClose={onClose}
    autoFocus={false}
    data-testid="modal-clients-update-pending"
  >
    <Stack direction="column" gap={2}>
      <Typography>{Messages.description1}</Typography>
      <Typography>{Messages.description2}</Typography>
      <Stack direction="row" justifyContent="flex-end" gap={1}>
        <Button
          variant="text"
          onClick={onClose}
          data-testid="modal-close-window-button"
        >
          {Messages.close}
        </Button>
        <Link href={PMM_HOME_URL} data-testid="modal-pmm-home-link">
          <Button variant="contained" data-testid="modal-pmm-home-button">
            {Messages.home}
          </Button>
        </Link>
      </Stack>
    </Stack>
  </Modal>
);
