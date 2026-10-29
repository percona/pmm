import type { ButtonProps } from '@mui/material/Button';
import type { ScopedThemeName } from 'components/theme-scope';

export interface TextSelectOption<T> {
  label: string;
  value: T;
}

export interface TextSelectProps<T> {
  value: T;
  label?: string;
  options: TextSelectOption<T>[];
  onChange: (value: T) => void;
  disabled?: boolean;
  disabledValue?: string;
  /** What the button shows for the selected value, when it differs from its menu label. */
  selectedLabel?: string;
  startIcon?: ButtonProps['startIcon'];
  buttonProps?: Omit<
    ButtonProps,
    'onClick' | 'variant' | 'endIcon' | 'startIcon' | 'disabled'
  >;
  'data-testid-button'?: string;
  /** Draws the button alone in another Peak theme; the menu keeps the page's. */
  buttonTheme?: ScopedThemeName;
  /** A menu as wide as its options, under the button's arrow, rather than the Menu surface's minimum width from its start. */
  compactMenu?: boolean;
}
