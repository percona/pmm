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
  // Which option the keyboard is on. MUI tracks this internally for its own selection, but
  // does not expose it to the option row, and the cluster toggle lives on the row.
  const [highlightedOption, setHighlightedOption] =
    useState<ServiceOption | null>(null);
  const services = 'sessions' in props ? props.sessions : props.services;
  const serviceOptions = useMemo(() => getServiceOptions(services), [services]);
  const selectedServices = useMemo(
    () => serviceOptions.filter((option) => serviceIds?.includes(option.id)),
    [serviceOptions, serviceIds]
  );
  // With singleTechnology the picker feeds one view of live queries, which
  // cannot mix engines, so the first pick fixes the technology and the others
  // are disabled until the selection is cleared.
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

  // A cluster row is toggled by ServiceOption's own onClick, which the keyboard never reaches:
  // MUI keeps focus on the input, so Enter runs the Autocomplete's own selection instead. That
  // hands the cluster to handleServiceChange, and getServiceIds keeps only options of type
  // 'service', so the cluster contributes nothing and the selection silently does not change.
  // Space does not select at all -- it types into the input. Both are handled here so the
  // keyboard reaches the same toggle the mouse does.
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
      // The option carries its own group: a cluster spanning technologies is
      // listed under each of them and has no serviceType to derive one from.
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
      // Only the one-line 'label' summary needs this: without it a selection wider than the
      // field pushes the text input and the clear button onto a second line and the control
      // grows into the toolbar. The 'tags' presentation renders a chip row that is meant to
      // wrap, so forcing nowrap on it would squeeze the input instead.
      sx={
        tagPresentation === 'label'
          ? { '& .MuiAutocomplete-inputRoot': { flexWrap: 'nowrap' } }
          : undefined
      }
    />
  );
};

export default ServicesAutocompleteInput;
