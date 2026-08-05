import Autocomplete from '@mui/material/Autocomplete';
import { FC, useMemo, useState } from 'react';
import {
  ServiceInput,
  ServiceOption as ServiceOptionComponent,
} from './components';
import {
  getClusterSelectionState,
  getServiceIds,
  getServiceOptions,
  toggleClusterServices,
} from './ServicesAutocompleteInput.utils';
import {
  ServiceOption,
  ServicesAutocompleteInputProps,
} from './ServicesAutocompleteInput.types';
import ServiceTags from './components/ServiceTags';
import { hasMixedTechnologies } from 'pages/rta/components/technology';

const ServicesAutocompleteInput: FC<ServicesAutocompleteInputProps> = ({
  disabled = false,
  serviceIds,
  onServiceIdsChange,
  inputProps,
  tagPresentation = 'label',
  'data-testid': testId,
  ...props
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const services = 'sessions' in props ? props.sessions : props.services;
  const serviceOptions = useMemo(() => getServiceOptions(services), [services]);
  // Each screen decides from its own list: the technology is only called out
  // when the services in this picker span more than one of them.
  const showTechnology = useMemo(
    () => hasMixedTechnologies(services.map((service) => service.serviceType)),
    [services]
  );
  const selectedServices = useMemo(
    () => serviceOptions.filter((option) => serviceIds?.includes(option.id)),
    [serviceOptions, serviceIds]
  );

  const handleServiceChange = (
    _event: React.SyntheticEvent,
    value: ServiceOption[]
  ) => {
    const serviceIds = getServiceIds(value);
    onServiceIdsChange(serviceIds);
  };

  const handleClusterToggle = (clusterName: string) => {
    const newSelection = toggleClusterServices(
      clusterName,
      serviceOptions,
      selectedServices
    );
    const serviceIds = getServiceIds(newSelection);
    onServiceIdsChange(serviceIds);
  };

  return (
    <Autocomplete
      multiple
      open={isOpen}
      onOpen={() => setIsOpen(true)}
      onClose={() => setIsOpen(false)}
      options={serviceOptions}
      value={selectedServices}
      onChange={handleServiceChange}
      getOptionLabel={(option) => option.label}
      isOptionEqualToValue={(option, value) => option.id === value.id}
      disableCloseOnSelect
      limitTags={2}
      renderInput={(params) => (
        <ServiceInput
          {...params}
          hasSelectedServices={selectedServices.length > 0}
          isOpen={isOpen}
          {...inputProps}
        />
      )}
      renderTags={(value, getTagProps) => (
        <ServiceTags
          value={value}
          getTagProps={getTagProps}
          tagPresentation={tagPresentation}
        />
      )}
      renderOption={(props, option, { selected }) => (
        <ServiceOptionComponent
          {...props}
          key={option.id}
          option={option}
          selected={selected}
          showTechnology={showTechnology}
          clusterSelectionState={
            option.type === 'cluster'
              ? getClusterSelectionState(
                  option.label,
                  serviceOptions,
                  selectedServices
                )
              : undefined
          }
          onClusterToggle={handleClusterToggle}
        />
      )}
      disabled={disabled || serviceOptions.length === 0}
      data-testid={testId}
    />
  );
};

export default ServicesAutocompleteInput;
