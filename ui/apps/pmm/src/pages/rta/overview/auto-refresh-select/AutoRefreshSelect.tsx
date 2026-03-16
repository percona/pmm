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
  50% { opacity: 1; transform: scale(1.1); }
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
      startIcon={
        isFetching ? (
          <Icon
            name="electric-bolt"
            color="primary"
            sx={{ animation: `${fadeInOut} 1.5s infinite` }}
          />
        ) : (
          <Icon name="electric-bolt-off" />
        )
      }
    />
  );
};

export default AutoRefreshSelect;
