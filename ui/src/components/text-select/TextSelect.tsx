import { ArrowDropDown, ArrowDropUp, Check } from '@mui/icons-material';
import {
  Button,
  ListItem,
  ListItemIcon,
  ListItemText,
  Menu,
  menuClasses,
  MenuItem,
  Stack,
} from '@mui/material';
import { MouseEvent, useState } from 'react';
import { TextSelectOption, TextSelectProps } from './TextSelect.types';
import { Messages } from './TextSelect.messages';

export const TextSelect = <T,>({
  value,
  label,
  options,
  onChange,
}: TextSelectProps<T>) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);
  const selected = options.find((option) => option.value === value);

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

  return (
    <Stack>
      <Button
        variant="text"
        onClick={handleOpen}
        endIcon={open ? <ArrowDropUp /> : <ArrowDropDown />}
        data-testid="text-select-button"
      >
        {label || Messages.label}: {selected?.label || Messages.empty}
      </Button>
      <Menu
        open={open}
        anchorEl={anchorEl}
        onClose={handleClose}
        sx={{
          [`.${menuClasses.list}`]: {
            p: 0,
            m: 0,
          },
        }}
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
            {/* Use div as component to not have nested li elements */}
            <ListItem component="div">
              {option.value === value && (
                <ListItemIcon>
                  <Check />
                </ListItemIcon>
              )}
              <ListItemText>{option.label}</ListItemText>
            </ListItem>
          </MenuItem>
        ))}
      </Menu>
    </Stack>
  );
};
