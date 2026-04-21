import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { Messages } from 'pages/settings/Settings.messages';
import { FC } from 'react';
import { useFormContext } from 'react-hook-form';

const SettingsSubmitButton: FC = () => {
  const {
    formState: { isDirty, errors },
  } = useFormContext();
  const { isPending } = useUpdateSettings();

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
        disabled={!isDirty || isPending || Object.keys(errors).length > 0}
        data-testid="ssh-key-submit"
        sx={{ alignSelf: 'flex-start' }}
      >
        {isPending ? Messages.applying : Messages.applyChanges}
      </Button>
    </Stack>
  );
};

export default SettingsSubmitButton;
