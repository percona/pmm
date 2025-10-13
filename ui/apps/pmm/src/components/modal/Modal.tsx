import {
  Stack,
  Modal as MuiModal,
  Typography,
  IconButton,
} from '@mui/material';
import { FC } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import { ModalProps } from './Modal.types';

export const Modal: FC<ModalProps> = ({
  children,
  title,
  subtitle,
  ...props
}) => (
  <MuiModal {...props}>
    <Stack
      sx={(theme) => ({
        position: 'absolute',
        top: '50%',
        left: '50%',
        transform: 'translate(-50%, -50%)',
        backgroundColor: theme.palette.background.paper,
        minWidth: 480,
        maxWidth: '80vw',
        minHeight: 250,
        borderRadius: 1,
        boxShadow: theme.shadows[24],
        border: 'none',
      })}
    >
      <Stack>
        <Stack
          sx={{
            p: 2,
            pb: 0,
          }}
        >
          <Stack direction="row" justifyContent="space-between">
            <Typography variant="h5">{title}</Typography>
            <IconButton
              sx={{
                p: 0,
              }}
              onClick={() =>
                props.onClose && props.onClose({}, 'escapeKeyDown')
              }
              data-testid="modal-close-button"
            >
              <CloseIcon />
            </IconButton>
          </Stack>
          {subtitle && (
            <Typography variant="body2" color="text.secondary">
              {subtitle}
            </Typography>
          )}
        </Stack>
        <Stack
          sx={{
            p: 2,
          }}
        >
          {children}
        </Stack>
      </Stack>
    </Stack>
  </MuiModal>
);
