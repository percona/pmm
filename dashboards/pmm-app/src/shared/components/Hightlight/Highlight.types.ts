import { ReactNode } from 'react';

export type ThemeMode = 'light' | 'dark';

export interface HighlightProps {
  children: ReactNode;
  language: string;
  className?: string;
  // Theme mode to render in when it differs from the page, such as inside an inverted tooltip
  mode?: ThemeMode;
}
