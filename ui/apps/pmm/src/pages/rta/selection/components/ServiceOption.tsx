import { FC, HTMLAttributes } from 'react';
import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import { ServiceOption as ServiceOptionType, ClusterSelectionState } from '../RealTimeSelectionForm.types';

interface ServiceOptionProps {
  option: ServiceOptionType;
  props: HTMLAttributes<HTMLLIElement>;
  selected: boolean;
  clusterSelectionState?: ClusterSelectionState;
  onClusterToggle?: (clusterName: string) => void;
}

export const ServiceOption: FC<ServiceOptionProps> = ({
  option,
  props,
  selected,
  clusterSelectionState,
  onClusterToggle,
}) => {
  const { key, ...otherProps } = props as HTMLAttributes<HTMLLIElement> & { key?: string };
  const isCluster = option.type === 'cluster';
  const isServiceInCluster = option.type === 'service' && Boolean(option.cluster);

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
