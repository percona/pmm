import { Controller, useFormContext } from "react-hook-form";
import { Checkbox as MUICheckbox } from "@mui/material";
import { CheckboxProps } from "./checkbox.types";
import { kebabize } from "@/utils";
import { LabeledContent } from "../../..";

const Checkbox = ({
  name,
  label,
  labelProps,
  control,
  controllerProps,
  checkboxProps,
  disabled,
}: CheckboxProps) => {
  const { control: contextControl } = useFormContext();

  const content = (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field }) => (
        <MUICheckbox
          {...field}
          checked={field.value}
          disabled={disabled}
          {...checkboxProps}
          inputProps={{
            // @ts-expect-error
            "data-testid": `checkbox-${kebabize(name)}`,
            ...checkboxProps?.inputProps,
          }}
        />
      )}
      {...controllerProps}
    />
  );

  return label ? (
    <LabeledContent label={label} {...labelProps}>
      {content}
    </LabeledContent>
  ) : (
    content
  );
};

export default Checkbox;
