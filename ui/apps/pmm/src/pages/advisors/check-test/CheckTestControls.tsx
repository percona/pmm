import { FC } from 'react';
import Autocomplete from '@mui/material/Autocomplete';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
import { Messages } from './CheckTest.messages';
import { CheckTestState } from './useCheckTest';

interface CheckTestControlsProps {
  test: CheckTestState;
  onTest: () => void;
  // extra disable on top of the "no service picked / already running" rules
  disabled?: boolean;
}

// service picker + Test button, rendered inside the overlay's footer toolbar
export const CheckTestControls: FC<CheckTestControlsProps> = ({
  test,
  onTest,
  disabled = false,
}) => (
  <>
    <Autocomplete
      // explicit id: MUI's auto-generated useId (":r1:") is not a valid CSS
      // identifier and breaks jsdom's selector matching
      id="advisor-check-test-service"
      size="small"
      sx={{ width: 280 }}
      options={test.serviceOptions}
      value={test.serviceOptions.find((o) => o.id === test.serviceId) ?? null}
      onChange={(_, option) => test.setServiceId(option?.id ?? null)}
      renderInput={(params) => (
        <TextField {...params} label={Messages.testService} />
      )}
      data-testid="advisor-check-form-test-service"
    />
    <Button
      variant="outlined"
      disabled={disabled || !test.serviceId || test.isTesting}
      onClick={onTest}
      data-testid="advisor-check-form-test"
    >
      {test.isTesting ? Messages.testing : Messages.test}
    </Button>
  </>
);
