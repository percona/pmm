import { FC, useState } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { enqueueSnackbar } from 'notistack';
import { useUser } from 'contexts/user';
import { Messages } from './RealtimeSelectionForm.messages';
import { useAvailableServices, useStartSessions } from 'hooks/api/useRealtime';
import { ServicesAutocompleteInput } from 'pages/rta/components/services-autocomplete-input';
import { RealtimeSession } from 'types/rta.types';

interface Props {
  onSuccess?: (sessions: RealtimeSession[]) => void;
}

const RealtimeSelectionForm: FC<Props> = ({ onSuccess }) => {
  const { user } = useUser();
  const [serviceIds, setServiceIds] = useState<string[]>([]);
  const { availableServices } = useAvailableServices();
  const startSessions = useStartSessions();

  const handleStart = async () => {
    await startSessions.mutateAsync(serviceIds, {
      onSuccess: (responses) => {
        enqueueSnackbar(Messages.startSuccess, { variant: 'success' });
        setServiceIds([]);
        onSuccess?.(responses.map((r) => r.session));
      },
      onError: (error) => {
        const message =
          error instanceof Error ? error.message : Messages.startError;
        enqueueSnackbar(message, { variant: 'error' });
      },
    });
  };

  return (
    <Stack gap={3} sx={{ width: '100%' }}>
      <ServicesAutocompleteInput
        tagPresentation="tags"
        services={availableServices}
        serviceIds={serviceIds}
        onServiceIdsChange={setServiceIds}
      />
      <Button
        data-testid="start-realtime-session"
        variant="contained"
        size="large"
        onClick={handleStart}
        disabled={
          serviceIds.length === 0 ||
          startSessions.isPending ||
          !user?.isPMMAdmin
        }
        sx={{
          borderRadius: 999,
          minHeight: 40,
          maxHeight: 40,
          textTransform: 'none',
          py: 0,
        }}
      >
        {Messages.startButton}
      </Button>
    </Stack>
  );
};

export default RealtimeSelectionForm;
