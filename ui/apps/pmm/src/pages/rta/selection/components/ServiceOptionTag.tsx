import { FC } from 'react';
import Box from '@mui/material/Box';
import Chip, { chipClasses } from '@mui/material/Chip';
import { AutocompleteRenderGetTagProps } from '@mui/material/Autocomplete';
import CloseIcon from '@mui/icons-material/Close';
import { ServiceOption } from '../form/RealTimeSelectionForm.types';

interface ServiceOptionTagProps {
  option: ServiceOption;
  tagProps: ReturnType<AutocompleteRenderGetTagProps>;
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
            backgroundColor: theme.palette.action.hover,
          })}
        >
          <CloseIcon
            sx={(theme) => ({
              fontSize: 12,
              color: theme.palette.text.secondary,
            })}
          />
        </Box>
      }
      {...restTagProps}
      sx={(theme) => ({
        height: 24,
        borderRadius: 12,
        backgroundColor: theme.palette.action.selected,
        px: 1,
        gap: 0.5,
        [`& .${chipClasses.label}`]: {
          px: 0,
          py: 0,
        },
        [`& .${chipClasses.deleteIcon}`]: {
          m: 0,
          p: 0,
        },
      })}
    />
  );
};
