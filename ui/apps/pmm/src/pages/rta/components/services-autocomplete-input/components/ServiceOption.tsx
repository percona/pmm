import { FC, HTMLAttributes } from 'react';
import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import {
  ClusterSelectionState,
  ServiceOption as ServiceOptionType,
} from '../ServicesAutocompleteInput.types';
interface Props extends HTMLAttributes<HTMLLIElement> {
  option: ServiceOptionType;
  selected: boolean;
  clusterSelectionState?: ClusterSelectionState;
  onClusterToggle?: (clusterName: string) => void;
  disabled?: boolean;
}

const ServiceOption: FC<Props> = ({
  option,
  selected,
  clusterSelectionState,
  onClusterToggle,
  disabled = false,
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

  // Cluster rows drive their own toggle rather than MUI's option click, so a
  // disabled row has to opt out of both handlers itself.
  const handleClick = isCluster
    ? (e: React.MouseEvent) => {
        e.stopPropagation();
        if (!disabled) {
          onClusterToggle?.(option.label);
        }
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
          ...(disabled && { opacity: 0.5, pointerEvents: 'none' }),
        },
      }}
    >
      <Checkbox
        checked={isCluster ? isFullySelected : selected}
        indeterminate={isPartiallySelected}
        disabled={disabled}
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
