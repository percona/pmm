import type { ReactNode } from 'react';
import type { BoxProps } from '@mui/material/Box';

export type CodeProps = Omit<BoxProps<'code'>, 'component' | 'children'> & {
  content: ReactNode;
};
