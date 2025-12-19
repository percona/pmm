import { ToggleButtonGroup } from "@mui/material";
import { kebabize } from "@/utils";
import LabeledContent from "../../../labeled-content";
import { Controller, useFormContext } from "react-hook-form";
import { ToggleButtonGroupInputProps } from "./toggle-button-group.types";

// TODO remove control prop from all inputs. We should just use useFormContext
const ToggleButtonGroupInput = ({
  name,
  label,
  controllerProps,
  labelProps,
  toggleButtonGroupProps = {},
  children,
}: ToggleButtonGroupInputProps) => {
  const { control, setValue } = useFormContext();
  const {
    sx: toggleButtonGroupSxProp,
    onChange: toggleButtonGroupOnChange = () => {},
    ...toggleButtonGroupRestProps
  } = toggleButtonGroupProps;
  const content = (
    <Controller
      name={name}
      control={control}
      render={({ field }) => (
        <ToggleButtonGroup
          {...field}
          fullWidth
          exclusive
          data-testid={`toggle-button-group-input-${kebabize(name)}`}
          sx={{
            "& > button": {
              flex: "1 1 0px",
            },
            ...toggleButtonGroupSxProp,
          }}
          onChange={(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            event: React.MouseEvent<HTMLElement> | any,
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            value: any
          ) => {
            if (value !== null) {
              const isNumber = typeof value === "number";
              if (isNumber) {
                event.target.valueAsNumber = value;
              } else {
                event.target.value = value;
              }

              toggleButtonGroupOnChange(
                event,
                isNumber ? event.target.valueAsNumber : event.target.value
              );
              setValue(name, value, { shouldTouch: true });
            }
          }}
          {...toggleButtonGroupRestProps}
        >
          {children}
        </ToggleButtonGroup>
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

export default ToggleButtonGroupInput;
