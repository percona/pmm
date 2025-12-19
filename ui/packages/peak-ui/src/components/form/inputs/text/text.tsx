import { TextField } from "@mui/material";
import { kebabize } from "@/utils";
import { Controller, useFormContext } from "react-hook-form";
import { TextInputProps } from "./text.types";

const TextInput = ({
  control,
  name,
  label,
  controllerProps,
  textFieldProps = {},
  isRequired,
  formHelperTextProps = {},
}: TextInputProps) => {
  const { control: contextControl } = useFormContext();
  const { sx: textFieldPropsSx, onChange, ...restFieldProps } = textFieldProps;
  const { sx: formHelperTextPropsSx } = formHelperTextProps;

  return (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field, fieldState: { error } }) => (
        <TextField
          label={label}
          {...field}
          size={restFieldProps?.size || "small"}
          sx={{ mt: 3, ...textFieldPropsSx }}
          {...restFieldProps}
          variant="outlined"
          required={isRequired}
          error={!!error}
          InputLabelProps={{
            shrink: true,
          }}
          inputProps={{
            "data-testid": `text-input-${kebabize(name)}`,
            onWheel: (e) => {
              (e.target as HTMLElement).blur();
            },
            onBlur: (event: React.FocusEvent<HTMLInputElement>) => {
              if (textFieldProps?.onBlur) {
                const modifiedEvent = {
                  ...event,
                  target: {
                    ...event.target,
                    value: textFieldProps.onBlur(event),
                  },
                };
                field.onChange(modifiedEvent);
              }
            },
            onChange: (event: React.ChangeEvent<HTMLInputElement>) => {
              if (onChange) {
                const modifiedEvent = {
                  ...event,
                  target: {
                    ...event.target,
                    value: onChange(event),
                  },
                };
                field.onChange(modifiedEvent);
              } else {
                field.onChange(event);
              }
            },
            ...restFieldProps?.inputProps,
          }}
          helperText={error ? error.message : restFieldProps?.helperText}
          FormHelperTextProps={{
            sx: formHelperTextPropsSx,
          }}
        />
      )}
      {...controllerProps}
    />
  );
};

export default TextInput;
