import { FC } from 'react';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import { SelectInput, TextInput } from '@percona/percona-ui';
import { useFieldArray, useFormContext } from 'react-hook-form';
import { FilterType } from 'types/alert-templates.types';
import { CreateRuleFormValues } from '../CreateAlertFromTemplate.types';
import { FILTER_TYPE_OPTIONS } from '../CreateAlertFromTemplate.constants';
import { Messages } from '../CreateAlertFromTemplate.messages';

export const FiltersSection: FC = () => {
  const { control } = useFormContext<CreateRuleFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: 'filters',
  });

  return (
    <Stack>
      <Typography variant="h6">{Messages.sections.filters}</Typography>
      {fields.map((field, index) => (
        <Stack key={field.id} direction="row" gap={2} alignItems="flex-start">
          <TextInput
            name={`filters.${index}.label`}
            label={Messages.filters.label}
          />
          <SelectInput
            name={`filters.${index}.type`}
            label={Messages.filters.type}
          >
            {FILTER_TYPE_OPTIONS.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </SelectInput>
          <TextInput
            name={`filters.${index}.regexp`}
            label={Messages.filters.regexp}
          />
          <IconButton
            aria-label={Messages.filters.remove}
            data-testid={`remove-filter-${index}`}
            onClick={() => remove(index)}
            sx={{ mt: 3 }}
          >
            <DeleteOutlineIcon />
          </IconButton>
        </Stack>
      ))}
      <Stack direction="row" mt={2}>
        <Button
          type="button"
          variant="text"
          startIcon={<AddOutlinedIcon />}
          data-testid="add-filter"
          onClick={() =>
            append({ type: FilterType.MATCH, label: '', regexp: '' })
          }
        >
          {Messages.filters.add}
        </Button>
      </Stack>
    </Stack>
  );
};

export default FiltersSection;
