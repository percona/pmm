import { FC, useEffect } from 'react';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle, TextInput } from '@percona/percona-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { z } from 'zod';
import { DEFAULT_INTERVAL } from '../CreateAlertFromTemplate.constants';
import { Messages } from '../CreateAlertFromTemplate.messages';
import { getIntervalError } from '../CreateAlertFromTemplate.utils';
import { EvaluationIntervalField } from './EvaluationIntervalField';

interface Props {
  open: boolean;
  onClose: () => void;
  onCreate: (group: string, interval: string) => void;
}

const schema = z.object({
  group: z
    .string()
    .trim()
    .min(1, { message: Messages.newGroupModal.nameRequired }),
  interval: z.string().superRefine((value, ctx) => {
    const error = getIntervalError(value);
    if (error) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: error });
    }
  }),
});

type NewGroupFormValues = z.infer<typeof schema>;

export const NewEvaluationGroupModal: FC<Props> = ({
  open,
  onClose,
  onCreate,
}) => {
  const methods = useForm<NewGroupFormValues>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: { group: '', interval: DEFAULT_INTERVAL },
  });

  useEffect(() => {
    if (open) {
      methods.reset({ group: '', interval: DEFAULT_INTERVAL });
    }
  }, [open, methods]);

  const onSubmit = (values: NewGroupFormValues) => {
    onCreate(values.group.trim(), values.interval);
  };

  return (
    <Dialog
      data-testid="new-eval-group-modal"
      open={open}
      onClose={onClose}
      maxWidth="xs"
      fullWidth
    >
      <DialogTitle onClose={onClose}>
        {Messages.newGroupModal.title}
      </DialogTitle>
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(onSubmit)}>
          <DialogContent>
            <Stack gap={2}>
              <DialogContentText>
                {Messages.newGroupModal.subtitle}
              </DialogContentText>
              <Stack gap={0.5}>
                <TextInput
                  name="group"
                  label={Messages.newGroupModal.nameLabel}
                  isRequired
                  textFieldProps={{
                    placeholder: Messages.newGroupModal.namePlaceholder,
                    slotProps: {
                      htmlInput: { 'data-testid': 'new-eval-group-name' },
                    },
                  }}
                />
                <Typography variant="caption" color="text.secondary">
                  {Messages.newGroupModal.nameDescription}
                </Typography>
              </Stack>
              <Stack gap={0.5}>
                <EvaluationIntervalField />
                <Typography variant="caption" color="text.secondary">
                  {Messages.newGroupModal.intervalDescription}
                </Typography>
              </Stack>
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button
              type="button"
              variant="text"
              data-testid="new-eval-group-cancel"
              onClick={onClose}
            >
              {Messages.newGroupModal.cancel}
            </Button>
            <Button
              type="submit"
              variant="contained"
              data-testid="new-eval-group-create"
              disabled={!methods.formState.isValid}
            >
              {Messages.newGroupModal.create}
            </Button>
          </DialogActions>
        </form>
      </FormProvider>
    </Dialog>
  );
};

export default NewEvaluationGroupModal;
