import type { SxProps, Theme } from '@mui/material/styles';
import { toggleButtonClasses } from '@mui/material/ToggleButton';
import { toggleButtonGroupClasses } from '@mui/material/ToggleButtonGroup';
import { FieldValues } from 'react-hook-form';
import { mergeSx } from '@/utils';
import { ToggleButtonGroupInputProps } from '../toggle-button-group';
import ToggleButtonGroupInput from '../toggle-button-group/toggle-button-group';

const { firstButton, middleButton, lastButton } = toggleButtonGroupClasses;
const { disabled } = toggleButtonClasses;

const withDisabled = (...classNames: string[]) =>
  classNames.flatMap((name) => [`& .${name}`, `& .${name}.${disabled}`]).join(', ');

const pillSx =
  (fullWidth: boolean): SxProps<Theme> =>
  (theme) => {
    const radius = `${theme.shape.borderRadiusFull}px`;

    return {
      ...(!fullWidth && { width: 'fit-content' }),
      [theme.breakpoints.down('sm')]: {
        flexDirection: 'row',
      },
      '& > button': {
        flex: '1 0 auto',
      },
      [withDisabled(middleButton, lastButton)]: {
        ml: 0,
        borderLeft: 0,
      },
      [withDisabled(firstButton, middleButton)]: {
        borderRight: 0,
      },
      [`& .${middleButton}`]: {
        borderRadius: 0,
      },
      [`& .${firstButton}`]: {
        borderRadius: `${radius} 0 0 ${radius}`,
      },
      [`& .${lastButton}`]: {
        borderRadius: `0 ${radius} ${radius} 0`,
      },
    };
  };

const ToggleButtonGroupInputRegular = <T extends FieldValues = FieldValues>({
  children,
  name,
  toggleButtonGroupProps,
  ...props
}: ToggleButtonGroupInputProps<T>) => {
  const { fullWidth = false, sx } = toggleButtonGroupProps ?? {};
  const isVertical = toggleButtonGroupProps?.orientation === 'vertical';

  return (
    <ToggleButtonGroupInput
      name={name}
      toggleButtonGroupProps={{
        ...toggleButtonGroupProps,
        fullWidth,
        sx: isVertical ? sx : mergeSx(pillSx(fullWidth), sx),
      }}
      {...props}
    >
      {children}
    </ToggleButtonGroupInput>
  );
};

export default ToggleButtonGroupInputRegular;
