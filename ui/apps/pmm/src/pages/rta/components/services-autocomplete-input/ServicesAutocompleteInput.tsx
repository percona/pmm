import Autocomplete from '@mui/material/Autocomplete';
import { FC, useMemo, useState } from 'react';
import {
  ServiceInput,
  ServiceOptionTag,
  ServiceOption as ServiceOptionComponent,
} from './components';
import {
  getClusterSelectionState,
  getServiceIds,
  getServiceOptions,
  toggleClusterServices,
} from './ServicesAutocompleteInput.utils';
import {
  PropsWithServices,
  PropsWithSessions,
  ServiceOption,
  ServicesAutocompleteInputProps,
} from './ServicesAutocompleteInput.types';

const ServicesAutocompleteInput: FC<ServicesAutocompleteInputProps> = ({
  disabled = false,
  serviceIds,
  onServiceIdsChange,
  inputProps,
  ...props
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const serviceOptions = useMemo(
    () =>
      'sessions' in props
        ? getServiceOptions(props.sessions)
        : getServiceOptions(props.services),
    [
      (props as PropsWithSessions).sessions,
      (props as PropsWithServices).services,
    ]
  );
  const selectedServices = useMemo(() => {
    return serviceOptions.filter((option) => serviceIds?.includes(option.id));
  }, [serviceOptions, serviceIds]);

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
      renderTags={(value, getTagProps) =>
        value
          .slice(0, 2)
          .map((option, index) => (
            <ServiceOptionTag
              {...getTagProps({ index })}
              key={option.id}
              option={option}
            />
          ))
      }
      renderOption={(props, option, { selected }) => (
        <ServiceOptionComponent
          {...props}
          key={option.id}
          option={option}
          selected={selected}
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
    />
  );
};

export default ServicesAutocompleteInput;
