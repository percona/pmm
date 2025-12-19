import { FormControlLabel, Switch, Typography } from "@mui/material";
import { kebabize } from "@/utils";
import { Controller, useFormContext } from "react-hook-form";
import { SwitchInputProps } from "./switch.types";

const SwitchInput = ({
  name,
  control,
  label,
  labelCaption,
  controllerProps,
  formControlLabelProps,
  switchFieldProps = {},
}: SwitchInputProps) => {
  const { control: contextControl } = useFormContext();
  const { onChange, ...restSwitchFieldProps } = switchFieldProps;
  return (
    <FormControlLabel
      label={
        <>
          <Typography variant="body1">{label}</Typography>
          {labelCaption && (
            <Typography variant="caption">{labelCaption}</Typography>
          )}
        </>
      }
      data-testid={`switch-input-${kebabize(name)}-label`}
      control={
        <Controller
          name={name}
          control={control ?? contextControl}
          render={({ field }) => (
            <Switch
              {...field}
              onChange={(e) => {
                onChange?.(e, e.target.checked);
                field.onChange(e);
              }}
              sx={{
                ...(labelCaption && {
                  alignSelf: "flex-start",
                  mt: -1,
                }),
              }}
              checked={field.value}
              data-testid={`switch-input-${kebabize(name)}`}
              {...restSwitchFieldProps}
            />
          )}
          {...controllerProps}
        />
      }
      {...formControlLabelProps}
    />
  );
};

export default SwitchInput;
