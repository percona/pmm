import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import { kebabize } from '@percona/peak-ui';
import { type FC, useState } from 'react';
import { TextField } from '../text-field';
import { Messages } from './PasswordField.messages';
import type { PasswordFieldProps } from './PasswordField.types';

export const PasswordField: FC<PasswordFieldProps> = (props) => {
  const [isRevealed, setIsRevealed] = useState(false);
  const toggleLabel = isRevealed ? Messages.hide : Messages.show;

  return (
    <TextField
      {...props}
      type={isRevealed ? 'text' : 'password'}
      endAdornment={
        <InputAdornment position="end">
          <IconButton
            edge="end"
            size="small"
            aria-label={toggleLabel}
            title={toggleLabel}
            data-testid={`${kebabize(props.name)}-reveal`}
            onClick={() => setIsRevealed((revealed) => !revealed)}
          >
            {isRevealed ? <VisibilityOffIcon /> : <VisibilityIcon />}
          </IconButton>
        </InputAdornment>
      }
    />
  );
};
