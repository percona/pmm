import DialogContent from '@mui/material/DialogContent';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle } from '@percona/ui-lib';
import { FC, useState } from 'react';
import { Messages } from './NewSessionModal.messages';
import Stack from '@mui/material/Stack';
import Link from '@mui/material/Link';
import Button from '@mui/material/Button';
import Autocomplete from '@mui/material/Autocomplete';
import TextField from '@mui/material/TextField';
import { useServices } from 'hooks/api/useServices';
import { ServiceType, VersionedService } from 'types/services.types';

interface Props {
  open: boolean;
  onClose: () => void;
  onCreateSession: (serviceIds: string[]) => Promise<void>;
}

// TODO: needs to reworked once selection pages is implemented
const NewSessionModal: FC<Props> = ({ open, onClose, onCreateSession }) => {
  const { data } = useServices(
    {
      serviceType: ServiceType.mongodb,
    },
    {
      enabled: open,
    }
  );
  const services = data?.mongodb || [];
  services.sort((a, b) => a.serviceName.localeCompare(b.serviceName));
  const [selectedServices, setSelectedServices] = useState<VersionedService[]>(
    []
  );

  const handleCreateSessions = async () => {
    await onCreateSession(selectedServices.map((service) => service.serviceId));
    setSelectedServices([]);
  };

  const handleClose = () => {
    setSelectedServices([]);
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose}>
      <DialogTitle onClose={handleClose}>{Messages.title}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        <Typography>{Messages.content}</Typography>
        {/* todo: use same input from selection page */}
        <Autocomplete
          multiple
          disableCloseOnSelect
          renderInput={(params) => (
            <TextField {...params} label={Messages.inputLabel} />
          )}
          options={services}
          getOptionLabel={(option: VersionedService) => option.serviceName}
          getOptionKey={(option: VersionedService) => option.serviceId}
          value={selectedServices}
          onChange={(_, value) => {
            setSelectedServices(value);
          }}
        />
        <Button
          disabled={selectedServices.length === 0}
          variant="contained"
          color="primary"
          onClick={handleCreateSessions}
          fullWidth
        >
          {Messages.actions.start}
        </Button>
        <Stack direction="column" gap={1}>
          <Typography variant="body2" color="text.secondary" textAlign="center">
            {Messages.disclaimer}
          </Typography>
          <Stack justifyContent="center" flexDirection="row" gap={2}>
            <Link
              variant="body2"
              href="#"
              rel="noopener noreferrer"
              target="_blank"
            >
              {Messages.documentation}
            </Link>
            <Link
              variant="body2"
              href="#"
              rel="noopener noreferrer"
              target="_blank"
            >
              {Messages.provideFeedback}
            </Link>
          </Stack>
        </Stack>
      </DialogContent>
    </Dialog>
  );
};
export default NewSessionModal;
