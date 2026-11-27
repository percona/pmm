import { ToggleButton, useTheme } from '@mui/material';
import { ToggleRegularButtonProps } from './toggle-regular-button.types';

const ToggleRegularButton = ({
  children,
  sx,
  dataTestId,
  ...props
}: ToggleRegularButtonProps) => {
  const theme = useTheme();

  return (
    <ToggleButton
      data-testid={dataTestId ?? 'toggle-button-group-btn'}
      disableRipple
      sx={{
        backgroundColor: 'background.default',
        color: theme.palette.text.primary,
        textTransform: 'none',
        borderStyle: 'solid',
        borderColor: theme.palette.primary.main,
        borderWidth: '2px',
        ':hover, &.Mui-selected:hover': {
          backgroundColor: theme.palette.action.hover,
          color: theme.palette.text.primary,
        },
        '&.Mui-selected, &.Mui-selected:hover': {
          backgroundColor: theme.palette.primary.main,
          color: 'background.default',
        },

        '&.MuiButtonBase-root': {
          wordWrap: 'break-word',
          whiteSpace: 'pre-wrap',
        },
        ...sx,
      }}
      {...props}
    >
      {children}
    </ToggleButton>
  );
};

export default ToggleRegularButton;
