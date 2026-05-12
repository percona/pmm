import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { Messages } from 'pages/settings/Settings.messages';
import { FC } from 'react';
import { useFormContext } from 'react-hook-form';

interface Props {
  testId: string;
}

const SettingsSubmitButton: FC<Props> = ({ testId }) => {
  const {
    formState: { isDirty, isSubmitting, isValid },
  } = useFormContext();

  return (
    <Stack
      sx={{
        position: 'sticky',
        bottom: 0,
        py: 2,
        bgcolor: 'background.paper',
        borderTop: 1,
        borderColor: 'divider',
        mt: 'auto',
        zIndex: 1,
        boxShadow: (theme) =>
          `-8px 0 0 0 ${theme.palette.background.paper}, 30px 0 0 0 ${theme.palette.background.paper}`,
      }}
    >
      <Button
        type="submit"
        variant="contained"
        disabled={!isValid || !isDirty || isSubmitting}
        data-testid={testId}
        sx={{ alignSelf: 'flex-start' }}
      >
        {isSubmitting ? Messages.applying : Messages.applyChanges}
      </Button>
    </Stack>
  );
};

export default SettingsSubmitButton;
