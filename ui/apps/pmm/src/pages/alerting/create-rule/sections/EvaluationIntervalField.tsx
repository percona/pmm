import { FC } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { TextInput } from '@percona/percona-ui';
import { useFormContext } from 'react-hook-form';
import { EVALUATION_INTERVAL_OPTIONS } from '../CreateAlertFromTemplate.constants';
import { Messages } from '../CreateAlertFromTemplate.messages';

// Used inside the new-evaluation-group modal (and reusable in any form that
// has an `interval` string field). Reads/writes the `interval` field via
// form context.
export const EvaluationIntervalField: FC = () => {
  const { watch, setValue } = useFormContext();
  const current = watch('interval');

  return (
    <Stack gap={1}>
      <TextInput
        name="interval"
        label={Messages.fields.interval}
        isRequired
        textFieldProps={{ helperText: Messages.intervalHelper }}
      />
      <Stack direction="row" gap={1} flexWrap="wrap">
        {EVALUATION_INTERVAL_OPTIONS.map((option) => (
          <Button
            key={option}
            type="button"
            size="small"
            variant={current === option ? 'contained' : 'outlined'}
            data-testid={`interval-preset-${option}`}
            onClick={() =>
              setValue('interval', option, {
                shouldValidate: true,
                shouldDirty: true,
              })
            }
          >
            {option}
          </Button>
        ))}
      </Stack>
    </Stack>
  );
};

export default EvaluationIntervalField;
