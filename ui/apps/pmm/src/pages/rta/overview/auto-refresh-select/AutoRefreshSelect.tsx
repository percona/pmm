import { Icon } from 'components/icon';
import { FC } from 'react';
import { Messages } from './AutoRefreshSelect.messages';
import { keyframes } from '@mui/material/styles';
import { TextSelect } from 'components/text-select';
import { REFRESH_INTERVAL_OPTIONS } from './AutoRefreshSelect.constants';

interface Props {
  isFetching: boolean;
  refreshInterval: number;
  onRefreshIntervalChange: (interval: number) => void;
}

const AutoRefreshSelect: FC<Props> = ({
  isFetching,
  refreshInterval,
  onRefreshIntervalChange,
}) => {
  const fadeInOut = keyframes`
  0% { opacity: 0; }
  40% { opacity: 1; }
  60% { opacity: 1; }
  100% { opacity: 0; }
`;

  return (
    <TextSelect
      value={refreshInterval}
      label={Messages.refreshInterval}
      options={REFRESH_INTERVAL_OPTIONS}
      onChange={onRefreshIntervalChange}
      disabled={!isFetching}
      disabledValue={Messages.off}
      data-testid-button="auto-refresh-button"
      buttonProps={{
        size: 'small',
        color: 'primary',
        disableElevation: true,
      }}
      startIcon={
        isFetching ? (
          <Icon
            name="electric-bolt"
            color="inherit"
            sx={{ animation: `${fadeInOut} 1.2s infinite` }}
          />
        ) : (
          <Icon name="electric-bolt-off" color="inherit" fontSize="inherit" />
        )
      }
    />
  );
};

export default AutoRefreshSelect;
