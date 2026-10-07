import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import type { FC } from 'react';
import { Link } from 'react-router-dom';
import { Messages } from './AddServiceActions.messages';
import type { AddServiceActionsProps } from './AddServiceActions.types';

export const AddServiceActions: FC<AddServiceActionsProps> = ({
  cancelTo,
  submitLabel,
  isSubmitting,
}) => (
  <Stack direction="row" justifyContent="flex-end" gap={1} sx={{ mt: 3 }}>
    <Button
      variant="outlined"
      component={Link}
      to={cancelTo}
      data-testid="add-service-cancel"
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
