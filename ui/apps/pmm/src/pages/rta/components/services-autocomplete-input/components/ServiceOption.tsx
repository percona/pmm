import { FC, HTMLAttributes } from 'react';
import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import {
  ClusterSelectionState,
  ServiceOption as ServiceOptionType,
} from '../ServicesAutocompleteInput.types';
import { Technology } from 'pages/rta/components/technology';

interface Props extends HTMLAttributes<HTMLLIElement> {
  option: ServiceOptionType;
  selected: boolean;
  clusterSelectionState?: ClusterSelectionState;
  onClusterToggle?: (clusterName: string) => void;
  showTechnology?: boolean;
}

const ServiceOption: FC<Props> = ({
  option,
  selected,
  clusterSelectionState,
  onClusterToggle,
  showTechnology = false,
  ...props
}) => {
  const { key, ...otherProps } = props as HTMLAttributes<HTMLLIElement> & {
    key?: string;
  };
  const isCluster = option.type === 'cluster';
  const isServiceInCluster =
    option.type === 'service' && Boolean(option.cluster);

  const isFullySelected = isCluster && clusterSelectionState === 'all';
  const isPartiallySelected = isCluster && clusterSelectionState === 'partial';

  const handleClick = isCluster
    ? (e: React.MouseEvent) => {
        e.stopPropagation();
        onClusterToggle?.(option.label);
      }
    : otherProps.onClick;

  return (
    <Box
      component="li"
      data-testid={'service-option-' + option.id}
      key={key}
      {...otherProps}
      onClick={handleClick}
      sx={{
        '&.MuiAutocomplete-option': {
          backgroundColor: 'transparent',
          minHeight: 40,
          padding: '0 8px',
          paddingLeft: isServiceInCluster ? '40px' : '8px',
          position: 'relative',
        },
      }}
    >
      <Checkbox
        checked={isCluster ? isFullySelected : selected}
        indeterminate={isPartiallySelected}
        size="small"
        sx={{ p: 1, mr: -0.5 }}
        onClick={
          isCluster
            ? (e) => {
                e.stopPropagation();
                onClusterToggle?.(option.label);
              }
            : undefined
        }
      />
      <Box
        sx={{
          flex: 1,
          py: '9px',
          px: 1,
          display: 'flex',
          alignItems: 'center',
          gap: 1,
        }}
      >
        {/* Fixed-width slot: an option we cannot label (a cluster whose services
            disagree) would otherwise sit flush left of its neighbours. */}
        {showTechnology && (
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              minWidth: 20,
            }}
          >
            <Technology serviceType={option.serviceType} iconOnly />
          </Box>
        )}
        {option.label}
      </Box>
      {isServiceInCluster && (
        <Box
          sx={{
            position: 'absolute',
            left: 28,
            top: 0,
            bottom: 0,
            width: 1,
            borderLeft: '1px solid',
            borderColor: 'divider',
          }}
        />
      )}
    </Box>
  );
};

export default ServiceOption;
