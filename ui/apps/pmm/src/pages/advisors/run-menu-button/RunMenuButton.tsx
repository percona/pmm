import { FC, useState, MouseEvent } from 'react';
import Button from '@mui/material/Button';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import ArrowDropDownOutlinedIcon from '@mui/icons-material/ArrowDropDownOutlined';

export interface RunMenuOption {
  label: string;
  // names of the enabled checks to run; empty disables the option
  names: string[];
}

interface RunMenuButtonProps {
  label: string;
  options: RunMenuOption[];
  disabled?: boolean;
  onRun: (option: RunMenuOption) => void;
}

export const RunMenuButton: FC<RunMenuButtonProps> = ({
  label,
  options,
  disabled,
  onRun,
}) => {
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);

  const handleOpen = (event: MouseEvent<HTMLElement>) =>
    setAnchorEl(event.currentTarget);

  const handleClose = () => setAnchorEl(null);

  const handleRun = (option: RunMenuOption) => {
    handleClose();
    onRun(option);
  };

  return (
    <>
      <Button
        endIcon={<ArrowDropDownOutlinedIcon />}
        disabled={disabled}
        onClick={handleOpen}
        data-testid={`run-menu-${label}`}
      >
        {label}
      </Button>
      <Menu anchorEl={anchorEl} open={!!anchorEl} onClose={handleClose}>
        {options.map((option) => (
          <MenuItem
            key={option.label}
            disabled={!option.names.length}
            onClick={() => handleRun(option)}
          >
            {option.label}
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};
