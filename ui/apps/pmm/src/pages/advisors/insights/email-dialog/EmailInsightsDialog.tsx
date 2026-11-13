import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import TextField from '@mui/material/TextField';
import { useEmailInsights } from 'hooks/api/useAdvisors';
import { enqueueSnackbar } from 'notistack';
import { Messages as EditorMessages } from 'pages/advisors/scheduler/schedule-editor/ScheduleEditor.messages';
import {
  emailListError,
  parseEmails,
} from 'pages/advisors/scheduler/schedule-editor/ScheduleEditor.utils';
import { type FC, type SyntheticEvent, useEffect, useState } from 'react';
import { type InsightsFilterParams } from 'types/advisors.types';
import { Messages } from '../AdvisorInsights.messages';

// mirrors maxEmailedInsights in managed/services/checks/notification.go
const MAX_EMAILED_INSIGHTS = 200;

interface EmailInsightsDialogProps {
  open: boolean;
  filters: InsightsFilterParams;
  filtered: boolean;
  // insights matching the filters, as the table counts them
  count: number;
  onClose: () => void;
}

export const EmailInsightsDialog: FC<EmailInsightsDialogProps> = ({
  open,
  filters,
  filtered,
  count,
  onClose,
}) => {
  const [text, setText] = useState('');
  // as in Settings, errors show once the field is edited, while Send waits for a valid list
  const [edited, setEdited] = useState(false);
  const { mutate: emailInsights, isPending } = useEmailInsights();

  useEffect(() => {
    if (open) {
      setText('');
      setEdited(false);
    }
  }, [open]);

  const recipients = parseEmails(text);
  const error = recipients.length
    ? emailListError(recipients)
    : Messages.email.required;
  const tooMany = count > MAX_EMAILED_INSIGHTS;

  // a request in flight can't be called back, so the dialog stays until it settles
  const close = () => {
    if (!isPending) {
      onClose();
    }
  };

  const send = (e?: SyntheticEvent) => {
    e?.preventDefault();
    if (error || tooMany || isPending) {
      return;
    }
    // a failure is reported by the API client, and the dialog stays open to retry
    emailInsights(
      { ...filters, emailAddresses: recipients },
      {
        onSuccess: ({ insightsCount }) => {
          enqueueSnackbar(
            Messages.email.sent(insightsCount, recipients.length),
            {
              variant: 'success',
            }
          );
          onClose();
        },
      }
    );
  };

  return (
    <Dialog
      open={open}
      onClose={close}
      fullWidth
      maxWidth="sm"
      data-testid="email-insights-dialog"
    >
      <form onSubmit={send} noValidate>
        <DialogTitle>{Messages.email.title}</DialogTitle>
        <DialogContent>
          <DialogContentText sx={{ mb: 2 }}>
            {filtered
              ? Messages.email.filtered(count)
              : Messages.email.unfiltered(count)}
          </DialogContentText>
          {tooMany && (
            <Alert
              severity="warning"
              sx={{ mb: 2 }}
              data-testid="email-insights-too-many"
            >
              {Messages.email.tooMany(count, MAX_EMAILED_INSIGHTS)}
            </Alert>
          )}
          <TextField
            // deterministic id: React's useId default breaks jsdom selector matching
            id="email-insights-addresses"
            label={EditorMessages.email.addresses}
            value={text}
            onChange={(e) => {
              setText(e.target.value);
              setEdited(true);
            }}
            // Enter sends, as in a single-line field; addresses never need a new line
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                send(e);
              }
            }}
            placeholder={EditorMessages.email.addressesPlaceholder}
            multiline
            minRows={2}
            fullWidth
            autoFocus
            error={edited && !!error}
            helperText={
              (edited && error) ||
              (recipients.length
                ? EditorMessages.email.addressesCount(recipients.length)
                : EditorMessages.email.addressesHelp)
            }
            slotProps={{
              htmlInput: { 'data-testid': 'email-insights-addresses' },
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button color="inherit" disabled={isPending} onClick={close}>
            {Messages.email.cancel}
          </Button>
          <Button
            type="submit"
            variant="contained"
            disabled={!!error || tooMany || isPending}
            data-testid="email-insights-send"
          >
            {Messages.email.send}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};
