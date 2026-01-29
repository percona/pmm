import { StackProps } from '@mui/material/Stack';
import { CodeLanguage } from 'types/util.types';

export interface CodeBlockProps {
  code: string;
  language: CodeLanguage;
  containerProps?: StackProps;
}
