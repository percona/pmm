import { ToggleButton, useTheme } from '@mui/material';
import { ToggleCardProps } from './toggle-card.types';

const ToggleCard = ({ children, sx, ...props }: ToggleCardProps) => {
  const theme = useTheme();

  return (
    <ToggleButton
      disableRipple
      sx={{
        backgroundColor: 'background.default',
        boxShadow: 4,
        color: theme.palette.text.primary,
        p: 2,
        textTransform: 'none',
        border: 'none',
        ':hover, &.Mui-selected:hover': {
          backgroundColor: 'action.hover',
        },
        '&.Mui-selected': {
          outlineStyle: 'solid',
          outlineWidth: '2px',
          outlineColor: theme.palette.action.outlinedBorder,
          backgroundColor: 'background.default',
        },
        '&.MuiToggleButtonGroup-grouped': {
          '&:not(:last-of-type)': {
            borderTopRightRadius: `${theme.shape.borderRadius}px`,
            borderBottomRightRadius: `${theme.shape.borderRadius}px`,

            [theme.breakpoints.down('sm')]: {
              mb: 1,
            },
            [theme.breakpoints.up('sm')]: {
              mr: 0.5,
            },
          },
          '&:not(:first-of-type)': {
            borderTopLeftRadius: `${theme.shape.borderRadius}px`,
            borderBottomLeftRadius: `${theme.shape.borderRadius}px`,
          },
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

export default ToggleCard;
