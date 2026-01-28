import { FC } from 'react';
import TextField from '@mui/material/TextField';
import { AutocompleteRenderInputParams } from '@mui/material/Autocomplete';
import { Messages } from '../RealtimeSelection.messages';

interface ServiceInputProps {
  params: AutocompleteRenderInputParams;
  hasSelectedServices: boolean;
  isOpen: boolean;
}

export const ServiceInput: FC<ServiceInputProps> = ({
  params,
  hasSelectedServices,
  isOpen,
}) => (
  <TextField
    {...params}
    label={hasSelectedServices || isOpen ? Messages.selectLabel : undefined}
    placeholder={!hasSelectedServices ? Messages.searchPlaceholder : ''}
    variant="outlined"
  />
);
