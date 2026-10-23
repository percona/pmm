import ArrowDropUp from '@mui/icons-material/ArrowDropUp';
import ArrowDropDown from '@mui/icons-material/ArrowDropDown';
import Check from '@mui/icons-material/Check';
import { type MouseEvent, useId, useState } from 'react';
import { TextSelectOption, TextSelectProps } from './TextSelect.types';
import { Messages } from './TextSelect.messages';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import Menu, { menuClasses } from '@mui/material/Menu';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import { ThemeScope } from 'components/theme-scope';

export const TextSelect = <T,>({
  value,
  label,
  options,
  onChange,
  disabled,
  disabledValue,
  selectedLabel,
  startIcon,
  buttonProps,
  'data-testid-button': dataTestIdButton,
  buttonTheme,
  compactMenu,
}: TextSelectProps<T>) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const menuId = useId();
  const open = Boolean(anchorEl);
  const selected = options.find((option) => option.value === value);
  const valueLabel = disabled
    ? disabledValue
    : (selectedLabel ?? selected?.label);

  const handleOpen = (event: MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleChange = (option: TextSelectOption<T>) => {
    setAnchorEl(null);
    onChange(option.value);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const button = (
    <Button
      {...buttonProps}
      variant="text"
      onClick={handleOpen}
      endIcon={open ? <ArrowDropUp /> : <ArrowDropDown />}
      aria-haspopup="menu"
      aria-expanded={open || undefined}
      aria-controls={open ? menuId : undefined}
      data-testid={dataTestIdButton || 'text-select-button'}
      disabled={disabled}
      startIcon={startIcon}
    >
      {label || Messages.label}: {valueLabel || Messages.empty}
    </Button>
  );

  return (
    <Stack>
      {buttonTheme ? (
        <ThemeScope theme={buttonTheme}>{button}</ThemeScope>
      ) : (
        button
      )}
      <Menu
        id={menuId}
        open={open}
        anchorEl={anchorEl}
        onClose={handleClose}
        {...(compactMenu && {
          anchorOrigin: { vertical: 'bottom', horizontal: 'right' },
          transformOrigin: { vertical: 'top', horizontal: 'right' },
        })}
        sx={
          compactMenu
            ? { [`.${menuClasses.paper}`]: { minWidth: 0 } }
            : undefined
        }
        data-testid="text-select-menu"
      >
        {options.map((option) => (
          <MenuItem
            key={option.value as React.Key}
            selected={option.value === value}
            onClick={() => handleChange(option)}
            color="text.secondary"
            data-testid={'text-select-option-' + option.value}
          >
            {option.value === value && (
              <ListItemIcon>
                <Check />
              </ListItemIcon>
            )}
            <ListItemText>{option.label}</ListItemText>
          </MenuItem>
        ))}
      </Menu>
    </Stack>
  );
};
