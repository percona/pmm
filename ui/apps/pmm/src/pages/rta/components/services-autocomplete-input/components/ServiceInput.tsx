import { FC } from 'react';
import TextField from '@mui/material/TextField';
import { AutocompleteRenderInputParams } from '@mui/material/Autocomplete';
import { Messages } from '../ServicesAutocompleteInput.messages';

interface Props extends AutocompleteRenderInputParams {
  hasSelectedServices: boolean;
  isOpen: boolean;
}

const ServiceInput: FC<Props> = ({ hasSelectedServices, isOpen, ...props }) => (
  <TextField
    name="service"
    data-testid="realtime-service-input"
    label={hasSelectedServices || isOpen ? Messages.selectLabel : undefined}
    placeholder={!hasSelectedServices ? Messages.searchPlaceholder : ''}
    variant="outlined"
    {...props}
  />
);

export default ServiceInput;
