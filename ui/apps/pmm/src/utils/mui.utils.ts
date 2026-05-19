import { FormHelperTextProps } from '@mui/material/FormHelperText';

export const helperTextTestId = (testId: string) =>
  ({ 'data-testid': testId }) as unknown as Partial<FormHelperTextProps>;
