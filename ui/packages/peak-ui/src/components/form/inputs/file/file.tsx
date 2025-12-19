import { IconButton } from '@mui/material';
import { TextField } from '@mui/material';
import UpgradeIcon from '@mui/icons-material/Upgrade';
import { TextFieldProps } from '@mui/material';
import { Controller, useFormContext } from 'react-hook-form';

type FileInputProps = {
  name: string;
  label: string;
  textFieldProps?: TextFieldProps;
  fileInputProps?: React.DetailedHTMLProps<
    React.InputHTMLAttributes<HTMLInputElement>,
    HTMLInputElement
  >;
};

const FileInput = ({
  name,
  label,
  textFieldProps = {},
  fileInputProps = {},
}: FileInputProps) => {
  const { control } = useFormContext();
  return (
    <Controller
      name={name}
      control={control}
      render={({ field, fieldState: { error } }) => (
        <TextField
          label={label}
          {...field}
          {...textFieldProps}
          value={
            field.value && field.value instanceof File ? field.value.name : ''
          }
          type="text"
          size="small"
          error={!!error}
          InputProps={{
            endAdornment: (
              <IconButton component="label">
                <UpgradeIcon fontSize="medium" />
                <input
                  style={{ display: 'none' }}
                  type="file"
                  hidden
                  onChange={(event) => {
                    const { files } = event.target;

                    if (files) {
                      const file = files[0];
                      field.onChange(file);
                    }
                  }}
                  {...fileInputProps}
                />
              </IconButton>
            ),
          }}
          helperText={error ? error.message : textFieldProps?.helperText}
        />
      )}
    />
  );
};

export default FileInput;
