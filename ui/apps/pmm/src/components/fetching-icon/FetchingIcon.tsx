import { FC } from 'react';
import CachedIcon from '@mui/icons-material/Cached';
import { FetchingIconProps } from './FetchingIcon.types';

export const FetchingIcon: FC<FetchingIconProps> = ({ isFetching }) => (
  <CachedIcon
    sx={
      isFetching
        ? {
            animation: 'spin 2s linear infinite',
            '@keyframes spin': {
              '0%': {
                transform: 'rotate(360deg)',
              },
              '100%': {
                transform: 'rotate(0deg)',
              },
            },
          }
        : {}
    }
  />
);
