import { type FC, useState } from 'react';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import { TextInput } from '@percona/peak-ui';
import { Messages } from './SecretField.messages';

interface Props {
  name: string;
  label: string;
  testId: string;
  helperText?: string;
  isRequired?: boolean;
}

/**
 * A credential field that starts masked and can be revealed.
 *
 * The values are pasted out of a support ticket, so a reveal is what lets an
 * operator check a long opaque string before submitting it. peak-ui ships no
 * password input — `TextInput` forwards `textFieldProps` to MUI's `TextField`,
 * which is where the adornment goes.
 */
export const SecretField: FC<Props> = ({
  name,
  label,
  testId,
  helperText,
  isRequired,
}) => {
  const [isRevealed, setIsRevealed] = useState(false);
  const toggleLabel = isRevealed ? Messages.hideSecret : Messages.revealSecret;

  return (
    <TextInput
      name={name}
      label={label}
      isRequired={isRequired}
      textFieldProps={{
        type: isRevealed ? 'text' : 'password',
        autoComplete: 'off',
        helperText,
        slotProps: {
          htmlInput: { 'data-testid': testId },
          input: {
            endAdornment: (
              <InputAdornment position="end">
                <IconButton
                  edge="end"
                  size="small"
                  aria-label={toggleLabel}
                  title={toggleLabel}
                  data-testid={`${testId}-reveal`}
                  onClick={() => setIsRevealed((revealed) => !revealed)}
                >
                  {isRevealed ? <VisibilityOffIcon /> : <VisibilityIcon />}
                </IconButton>
              </InputAdornment>
            ),
          },
        },
      }}
    />
  );
};
