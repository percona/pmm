import Typography from '@mui/material/Typography';
import { Modal } from 'components/modal';
import { ModalProps } from 'components/modal/Modal.types';
import { FC } from 'react';
import { Messages } from './RunningAgentsModal.messages';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import AddIcon from '@mui/icons-material/Add';

type Props = Pick<ModalProps, 'open' | 'onClose'>;

// Placeholder for the running agents modal
const RunningAgentsModal: FC<Props> = ({ open, onClose }) => (
  <Modal
    data-testid="running-agents-modal"
    open={open}
    title={Messages.title}
    onClose={onClose}
  >
    <Stack gap={2} direction="column">
      <Typography variant="body2" color="text.secondary">
        {Messages.description}
      </Typography>
      <Stack direction="row" justifyContent="space-between" gap={2}>
        <Button
          data-testid="running-agents-modal-add-another-button"
          variant="text"
          color="primary"
          startIcon={<AddIcon />}
        >
          {Messages.actions.addAnother}
        </Button>
        <Stack direction="row" gap={2}>
          <Button
            variant="text"
            color="primary"
            data-testid="running-agents-modal-close-window-button"
            onClick={() => onClose?.({}, 'escapeKeyDown')}
          >
            {Messages.actions.closeWindow}
          </Button>
          <Button
            data-testid="running-agents-modal-stop-all-agents-button"
            variant="contained"
            color="primary"
          >
            {Messages.actions.stopAllAgents}
          </Button>
        </Stack>
      </Stack>
    </Stack>
  </Modal>
);

export default RunningAgentsModal;
