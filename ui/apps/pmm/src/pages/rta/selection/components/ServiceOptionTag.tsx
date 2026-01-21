import { FC } from 'react';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import CloseIcon from '@mui/icons-material/Close';
import { ServiceOption } from '../RealTimeSelectionForm.utils';

interface ServiceOptionTagProps {
  option: ServiceOption;
  tagProps: {
    key: number;
    className: string;
    disabled: boolean;
    'data-tag-index': number;
    tabIndex: -1;
    onDelete: (event: React.MouseEvent<HTMLElement>) => void;
  };
}

export const ServiceOptionTag: FC<ServiceOptionTagProps> = ({
  option,
  tagProps,
}) => {
  const { key, ...restTagProps } = tagProps;

  return (
    <Chip
      key={key}
      label={option.label}
      deleteIcon={
        <Box
          sx={(theme) => ({
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 16,
            height: 16,
            borderRadius: '50%',
            backgroundColor:
              theme.palette.mode === 'dark'
                ? 'rgba(255, 255, 255, 0.2)'
                : 'rgba(0, 0, 0, 0.15)',
          })}
        >
          <CloseIcon
            sx={(theme) => ({
              fontSize: 12,
              color:
                theme.palette.mode === 'dark'
                  ? 'rgba(255, 255, 255, 0.9)'
                  : 'rgba(0, 0, 0, 0.6)',
            })}
          />
        </Box>
      }
      {...restTagProps}
      sx={(theme) => ({
        height: 24,
        borderRadius: 12,
        backgroundColor:
          theme.palette.mode === 'dark'
            ? 'rgba(255, 255, 255, 0.16)'
            : 'rgba(0, 0, 0, 0.08)',
        px: 1,
        gap: 0.5,
        '& .MuiChip-label': {
          px: 0,
          py: 0,
        },
        '& .MuiChip-deleteIcon': {
          m: 0,
          p: 0,
        },
      })}
    />
  );
};
