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
  isServiceOptionDisabled,
  toggleClusterServices,
} from './ServicesAutocompleteInput.utils';
import {
  ServiceOption,
  ServicesAutocompleteInputProps,
} from './ServicesAutocompleteInput.types';
import ServiceTags from './components/ServiceTags';
import { sharedTechnology } from 'pages/rta/components/technology';

const ServicesAutocompleteInput: FC<ServicesAutocompleteInputProps> = ({
  disabled = false,
  serviceIds,
  onServiceIdsChange,
  inputProps,
  tagPresentation = 'label',
  singleTechnology = false,
  'data-testid': testId,
  ...props
}) => {
  const [isOpen, setIsOpen] = useState(false);
  // MUI does not expose its highlighted option to the row, where the cluster toggle lives.
  const [highlightedOption, setHighlightedOption] =
    useState<ServiceOption | null>(null);
  const services = 'sessions' in props ? props.sessions : props.services;
  const serviceOptions = useMemo(() => getServiceOptions(services), [services]);
  const selectedServices = useMemo(
    () => serviceOptions.filter((option) => serviceIds?.includes(option.id)),
    [serviceOptions, serviceIds]
  );
  // One view of live queries cannot mix engines, so the first pick fixes the technology.
  const selectedTechnology = useMemo(
    () =>
      singleTechnology
        ? sharedTechnology(
            selectedServices
              .filter((option) => option.type === 'service')
              .map((option) => option.serviceType)
          )
        : undefined,
    [singleTechnology, selectedServices]
  );
  const isOptionDisabled = (option: ServiceOption) =>
    isServiceOptionDisabled(option, selectedTechnology, singleTechnology);

  const handleServiceChange = (
    _event: React.SyntheticEvent,
    value: ServiceOption[]
  ) => {
    const serviceIds = getServiceIds(value);
    onServiceIdsChange(serviceIds);
  };

  const handleClusterToggle = (clusterOption: ServiceOption) => {
    const newSelection = toggleClusterServices(
      clusterOption,
      serviceOptions,
      selectedServices
    );
    const serviceIds = getServiceIds(newSelection);
    onServiceIdsChange(serviceIds);
  };

  // The keyboard never reaches the cluster row's onClick: MUI's Enter selection drops cluster
  // options (see getServiceIds), and Space types into the input.
  const handleKeyDown = (
    event: React.KeyboardEvent & { defaultMuiPrevented?: boolean }
  ) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }

    if (
      !highlightedOption ||
      highlightedOption.type !== 'cluster' ||
      isOptionDisabled(highlightedOption)
    ) {
      return;
    }

    // Stops MUI from also running its selection for Enter and from inserting a space.
    event.defaultMuiPrevented = true;
    event.preventDefault();

    handleClusterToggle(highlightedOption);
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
      onHighlightChange={(_event, option) => setHighlightedOption(option)}
      onKeyDown={handleKeyDown}
      getOptionLabel={(option) => option.label}
      // A cluster spanning technologies has no serviceType to derive its group from.
      groupBy={(option) => option.technology}
      getOptionDisabled={isOptionDisabled}
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
          disabled={isOptionDisabled(option)}
          clusterSelectionState={
            option.type === 'cluster'
              ? getClusterSelectionState(
                  option,
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
      // Keeps the one-line 'label' summary from growing into the toolbar; the 'tags' chip row
      // is meant to wrap.
      sx={
        tagPresentation === 'label'
          ? { '& .MuiAutocomplete-inputRoot': { flexWrap: 'nowrap' } }
          : undefined
      }
    />
  );
};

export default ServicesAutocompleteInput;
