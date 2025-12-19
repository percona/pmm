import { ToggleButtonGroupInputProps } from '../toggle-button-group';
import ToggleButtonGroupInput from '../toggle-button-group/toggle-button-group';

const ToggleButtonGroupInputRegular = ({
  children,
  name,
  toggleButtonGroupProps,
  ...props
}: ToggleButtonGroupInputProps) => {
  return (
    <ToggleButtonGroupInput
      name={name}
      toggleButtonGroupProps={{
        sx: {
          width: 'fit-content',
          '.MuiToggleButtonGroup-middleButton, .MuiToggleButtonGroup-lastButton':
            {
              ml: 0,
              borderLeft: 0,
            },

          '.MuiButtonBase-root:not(:first-child, :last-child)': {
            borderRadius: '0',
          },

          '.MuiButtonBase-root: first-child': {
            borderTopLeftRadius: '128px',
            borderBottomLeftRadius: '128px',
            borderTopRightRadius: '0%',
            borderBottomRightRadius: '0%',
          },
          '.MuiButtonBase-root: last-child': {
            borderTopLeftRadius: '0',
            borderBottomLeftRadius: '0',
            borderTopRightRadius: '128px',
            borderBottomRightRadius: '128px',
          },
        },
        ...toggleButtonGroupProps,
      }}
      {...props}
    >
      {children}
    </ToggleButtonGroupInput>
  );
};

export default ToggleButtonGroupInputRegular;
