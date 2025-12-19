import { Box, Button, IconButton } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import { FieldError, useFieldArray, useFormContext } from 'react-hook-form';
import TextInput from '../text';

interface MultipleTextInputProps {
  fieldName: string;
  onRemove?: (nrOfFields: number) => void;
  onChange?: (
    nrOfFields: number,
    index: number,
    field: 'key' | 'value'
  ) => void;
}

const MultipleTextInput = ({
  fieldName,
  onRemove,
  onChange,
}: MultipleTextInputProps) => {
  const {
    control,
    formState: { errors },
    watch,
  } = useFormContext();
  const { fields, append, remove } = useFieldArray({
    control,
    name: fieldName,
  });

  const error = (index: number, key: 'key' | 'value'): FieldError | undefined =>
    // @ts-ignore
    errors?.[fieldName]?.[index]?.[key];

  const handleAdd = () => {
    append({ key: '', value: '' });
  };

  const handleOnChange = async (
    value: string,
    index: number,
    field: 'key' | 'value'
  ) => {
    onChange?.(fields.length, index, field);
    return value;
  };

  const handleOnRemove = (index: number) => {
    remove(index);
    onRemove?.(fields.length);
  };

  const fieldValues = watch(fieldName) || [];
  const hasEmptyFields = fieldValues.some(
    (field: { key: string; value: string }) => {
      return !field.key && !field.value;
    }
  );

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
