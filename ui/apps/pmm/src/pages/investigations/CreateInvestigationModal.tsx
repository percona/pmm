import {
  Button,
  DialogActions,
  DialogContent,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
} from '@mui/material';
import { Dialog, DialogTitle } from '@percona/ui-lib';
import { FC, useEffect, useState } from 'react';
import type { CreateInvestigationBody } from 'api/investigations';

export interface CreateInvestigationModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (body: CreateInvestigationBody) => void;
  isPending?: boolean;
  /** Prefill from URL params or alert context */
  initial?: Partial<CreateInvestigationBody>;
}

export const CreateInvestigationModal: FC<CreateInvestigationModalProps> = ({
  open,
  onClose,
  onSubmit,
  isPending = false,
  initial,
}) => {
  const [title, setTitle] = useState(initial?.title ?? '');
  const [sourceType, setSourceType] = useState<string>(initial?.sourceType ?? 'manual');
  const [sourceRef, setSourceRef] = useState(initial?.sourceRef ?? '');
  const [timeFrom, setTimeFrom] = useState(initial?.timeFrom ?? '');
  const [timeTo, setTimeTo] = useState(initial?.timeTo ?? '');

  useEffect(() => {
    if (open) {
      setTitle(initial?.title ?? '');
      setSourceType(initial?.sourceType ?? 'manual');
      setSourceRef(initial?.sourceRef ?? '');
      setTimeFrom(initial?.timeFrom ?? '');
      setTimeTo(initial?.timeTo ?? '');
    }
  }, [open, initial?.title, initial?.sourceType, initial?.sourceRef, initial?.timeFrom, initial?.timeTo]);

  const handleSubmit = () => {
    const body: CreateInvestigationBody = {
      title: title.trim() || 'New investigation',
      sourceType: sourceType === 'manual' ? undefined : sourceType,
      sourceRef: sourceRef.trim() || undefined,
      timeFrom: timeFrom.trim() || undefined,
      timeTo: timeTo.trim() || undefined,
    };
    onSubmit(body);
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      aria-labelledby="create-investigation-modal-title"
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle id="create-investigation-modal-title" onClose={onClose}>
        Create investigation
      </DialogTitle>
      <DialogContent>
        <Stack gap={2} sx={{ mt: 0.5 }}>
          <TextField
            label="Title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="What to investigate"
            size="small"
            fullWidth
          />
          <FormControl size="small" fullWidth>
            <InputLabel>Source type</InputLabel>
            <Select
              value={sourceType}
              label="Source type"
              onChange={(e) => setSourceType(e.target.value)}
            >
              <MenuItem value="manual">Manual</MenuItem>
              <MenuItem value="alert">Alert</MenuItem>
            </Select>
          </FormControl>
          {sourceType === 'alert' && (
            <TextField
              label="Alert reference"
              value={sourceRef}
              onChange={(e) => setSourceRef(e.target.value)}
              placeholder="Fingerprint or alert ID"
              size="small"
              fullWidth
            />
          )}
          <TextField
            label="Time from"
            value={timeFrom}
            onChange={(e) => setTimeFrom(e.target.value)}
            placeholder="e.g. 2025-01-01T00:00:00Z"
            size="small"
            fullWidth
          />
          <TextField
            label="Time to"
            value={timeTo}
            onChange={(e) => setTimeTo(e.target.value)}
            placeholder="e.g. 2025-01-01T23:59:59Z"
            size="small"
            fullWidth
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleSubmit}
          disabled={isPending}
        >
          {isPending ? 'Creating...' : 'Create'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
