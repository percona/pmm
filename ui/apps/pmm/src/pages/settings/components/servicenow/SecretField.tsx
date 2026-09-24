import { FC, useState } from 'react';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import { TextInput } from '@percona/peak-ui';
import { Messages } from '../../Settings.messages';

interface Props {
  name: string;
  label: string;
  helperText: string;
  testId: string;
}

/**
 * A credential field that starts masked and can be revealed.
 *
 * The values are pasted out of a support ticket, so a reveal is what lets an
 * operator check a long opaque string before submitting it. peak-ui ships no
 * password input — `TextInput` forwards `textFieldProps` to MUI's `TextField`,
 * which is where the adornment goes.
 */
export const SecretField: FC<Props> = ({ name, label, helperText, testId }) => {
  const [isRevealed, setIsRevealed] = useState(false);
  const { revealSecret, hideSecret } = Messages.serviceNow;
  const toggleLabel = isRevealed ? hideSecret : revealSecret;

  return (
    <TextInput
      name={name}
      label={label}
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
