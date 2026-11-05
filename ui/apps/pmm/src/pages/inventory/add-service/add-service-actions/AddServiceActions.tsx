import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import type { FC } from 'react';
import { Link } from 'react-router-dom';
import { STICKY_ACTIONS_SX } from './AddServiceActions.constants';
import { Messages } from './AddServiceActions.messages';
import type { AddServiceActionsProps } from './AddServiceActions.types';

export const AddServiceActions: FC<AddServiceActionsProps> = ({
  cancelTo,
  onCancel,
  submitLabel,
  isSubmitting,
  isSticky,
}) => (
  <Stack
    direction="row"
    justifyContent="flex-end"
    gap={1}
    sx={isSticky ? STICKY_ACTIONS_SX : undefined}
  >
    <Button
      variant="outlined"
      data-testid="add-service-cancel"
      {...(onCancel
        ? { onClick: onCancel }
        : { component: Link, to: cancelTo })}
    >
      {Messages.cancel}
    </Button>
    {submitLabel && (
      <Button
        type="submit"
        variant="contained"
        disabled={isSubmitting}
        data-testid="add-service-submit"
      >
        {submitLabel}
      </Button>
    )}
  </Stack>
);
