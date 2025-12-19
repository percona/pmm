import { Controller, useFormContext } from "react-hook-form";
import { DateTimePicker } from "@mui/x-date-pickers/DateTimePicker";
import { kebabize } from "@/utils";
import { DateTimePickerInputProps } from "./date-time-picker.types";
import { PickerValidDate } from "@mui/x-date-pickers";

const DateTimePickerInput = <T extends PickerValidDate>({
  name,
  control,
  controllerProps,
  ...dateTimePickerProps
}: DateTimePickerInputProps<T>) => {
  const { control: contextControl } = useFormContext();

  return (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field, fieldState: { error } }) => (
        <DateTimePicker
          {...field}
          inputRef={field.ref}
          slotProps={{
            textField: {
              error: !!error,
              helperText: error ? error.message : "",
              inputProps: {
                "data-testid": `date-time-picker-${kebabize(name)}`,
              },
            },
          }}
          {...dateTimePickerProps}
        />
      )}
      {...controllerProps}
    />
  );
};

export default DateTimePickerInput;
