import { Box, Button, IconButton } from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import { DeleteOutlineOutlined as DeleteOutlineOutlinedIcon } from '@mui/icons-material';
import { Control, FieldError, useFieldArray, useFormContext } from 'react-hook-form';
import TextInput from '../text';

interface MultipleTextInputProps {
  /** Form field holding the list, as an array of `{ key, value }` rows. */
  fieldName: string;
  /** react-hook-form control; outside a `FormProvider` the rows still save but validation errors are not shown. */
  control?: Control;
  /** Called with the row count before removal after the user deletes a row. */
  onRemove?: (nrOfFields: number) => void;
  /** Called when the user edits a row, with the row count, the row index, and whether the key or the value changed. */
  onChange?: (nrOfFields: number, index: number, field: 'key' | 'value') => void;
}

const MultipleTextInput = ({
  fieldName,
  control: controlProp,
  onRemove,
  onChange,
}: MultipleTextInputProps) => {
  const formContext = useFormContext();
  const control = controlProp ?? formContext?.control;
  const errors = formContext?.formState?.errors;
  const watch = formContext?.watch;

  const { fields, append, remove } = useFieldArray({
    control,
    name: fieldName,
  });

  const error = (index: number, key: 'key' | 'value'): FieldError | undefined =>
    // @ts-expect-error react-hook-form errors path is not strongly typed for nested field arrays
    errors?.[fieldName]?.[index]?.[key];

  const handleAdd = () => {
    append({ key: '', value: '' });
  };

  const handleOnChange = async (value: string, index: number, field: 'key' | 'value') => {
    onChange?.(fields.length, index, field);
    return value;
  };

  const handleOnRemove = (index: number) => {
    remove(index);
    onRemove?.(fields.length);
  };

  const fieldValues = watch?.(fieldName) || [];
  const hasEmptyFields = fieldValues.some((field: { key: string; value: string }) => {
    return !field.key && !field.value;
  });

  return (
    <Box
      sx={{
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        gap: 2,
        pt: 2,
      }}
    >
      {fields.map((field, index) => (
        <Box
          key={field.id}
          sx={{
            width: '100%',
            display: 'flex',
            gap: 1,
            alignItems: 'flex-start',
            justifyContent: 'space-between',
          }}
        >
          <TextInput
            name={`${fieldName}.${index}.key`}
            control={control}
            textFieldProps={{
              variant: 'outlined',
              placeholder: 'Enter key',
              error: !!error(index, 'key'),
              helperText: error(index, 'key')?.message,
              sx: {
                width: '100%',
                mt: 0,
              },
              onChange: (e) => {
                handleOnChange(e.target.value, index, 'key');
              },
            }}
          />
          <TextInput
            name={`${fieldName}.${index}.value`}
            control={control}
            textFieldProps={{
              variant: 'outlined',
              placeholder: 'Enter value',
              error: !!error(index, 'value'),
              helperText: error(index, 'value')?.message,
              sx: {
                width: '100%',
                mt: 0,
              },
              onChange: (e) => {
                handleOnChange(e.target.value, index, 'value');
              },
            }}
          />
          <IconButton
            onClick={() => {
              handleOnRemove(index);
            }}
          >
            <DeleteOutlineOutlinedIcon />
          </IconButton>
        </Box>
      ))}

      <Button
        variant="text"
        size="small"
        startIcon={<AddIcon />}
        onClick={handleAdd}
        disabled={hasEmptyFields}
      >
        Add new
      </Button>
    </Box>
  );
};

export default MultipleTextInput;
