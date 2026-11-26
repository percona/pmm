import type { FC } from 'react';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import { AutoCompleteInput, TextInput } from '@percona/peak-ui';
import { useFieldArray, useFormContext, useWatch } from 'react-hook-form';
import { Messages } from './AdvisorCheckForm.messages';
import { QUERY_PARAMETERS_BY_TYPE } from './AdvisorCheckForm.constants';
import type { AdvisorCheckFormValues } from './AdvisorCheckForm.schema';

interface QueryParametersInputProps {
  queryIndex: number;
}

// name/value rows of one query's parameters
export const QueryParametersInput: FC<QueryParametersInputProps> = ({
  queryIndex,
}) => {
  const { control } = useFormContext<AdvisorCheckFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: `queries.${queryIndex}.parameters`,
  });
  const type = useWatch({ control, name: `queries.${queryIndex}.type` });
  const nameOptions = QUERY_PARAMETERS_BY_TYPE[type] ?? [];
  const testId = `check-query-${queryIndex}-parameter`;

  return (
    <Stack gap={1}>
      {fields.map((field, index) => (
        <Stack
          key={field.id}
          direction="row"
          gap={2}
          alignItems="flex-start"
          data-testid={`${testId}-${index}`}
        >
          <AutoCompleteInput
            name={`queries.${queryIndex}.parameters.${index}.name`}
            label={Messages.fields.parameterName}
            options={nameOptions}
            autoCompleteProps={{
              // known names are suggestions; autoSelect commits a typed-in
              // name on blur, like the category field
              freeSolo: true,
              autoSelect: true,
              sx: { mt: 0, width: 200 },
            }}
          />
          <TextInput
            name={`queries.${queryIndex}.parameters.${index}.value`}
            label={Messages.fields.parameterValue}
            textFieldProps={{
              sx: { width: 200 },
              slotProps: {
                htmlInput: { 'data-testid': `${testId}-${index}-value` },
              },
            }}
          />
          <Tooltip title={Messages.removeParameter} arrow>
            <IconButton
              aria-label={Messages.removeParameter}
              onClick={() => remove(index)}
              data-testid={`${testId}-${index}-remove`}
            >
              <DeleteOutlineOutlinedIcon />
            </IconButton>
          </Tooltip>
        </Stack>
      ))}
      <Button
        size="small"
        startIcon={<AddOutlinedIcon />}
        onClick={() => append({ name: '', value: '' })}
        sx={{ alignSelf: 'flex-start' }}
        data-testid={`${testId}-add`}
      >
        {Messages.addParameter}
      </Button>
    </Stack>
  );
};
