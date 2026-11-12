import { FieldValues } from 'react-hook-form';
import { ToggleButtonGroupInputProps } from '../toggle-button-group';
import ToggleButtonGroupInput from '../toggle-button-group/toggle-button-group';

const ToggleButtonGroupInputRegular = <T extends FieldValues = FieldValues>({
  children,
  name,
  toggleButtonGroupProps,
  ...props
}: ToggleButtonGroupInputProps<T>) => {
  return (
    <ToggleButtonGroupInput
      name={name}
      toggleButtonGroupProps={{
        sx: (theme) => ({
          width: 'fit-content',
          '.MuiToggleButtonGroup-middleButton, .MuiToggleButtonGroup-lastButton': {
            ml: 0,
            borderLeft: 0,
          },

          '.MuiButtonBase-root:not(:first-child, :last-child)': {
            borderRadius: '0',
          },

          '.MuiButtonBase-root: first-child': {
            borderTopLeftRadius: `${theme.shape.borderRadiusFull}px`,
            borderBottomLeftRadius: `${theme.shape.borderRadiusFull}px`,
            borderTopRightRadius: '0%',
            borderBottomRightRadius: '0%',
          },
          '.MuiButtonBase-root: last-child': {
            borderTopLeftRadius: '0',
            borderBottomLeftRadius: '0',
            borderTopRightRadius: `${theme.shape.borderRadiusFull}px`,
            borderBottomRightRadius: `${theme.shape.borderRadiusFull}px`,
          },
        }),
        ...toggleButtonGroupProps,
      }}
      {...props}
    >
      {children}
    </ToggleButtonGroupInput>
  );
};

export default ToggleButtonGroupInputRegular;
