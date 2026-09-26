import SvgIcon from '@mui/material/SvgIcon';

export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

export type SvgIconComponent = typeof SvgIcon;

export type EmptyResponse = Record<string, never>;

export type CodeLanguage = 'text' | 'mongodb' | 'json' | 'sql';
