import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { Messages } from '../QanHeader.messages';
import { FC } from 'react';
import { Link } from 'react-router-dom';
import { useIsRealtimeQan } from 'hooks/utils/useLocation';
import { useSettings } from 'contexts/settings';
import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';

const QanHeaderTabs: FC = () => {
  const isRealtime = useIsRealtimeQan();
  const { settings } = useSettings();
  const historicalTo = settings?.nativeQanEnabled
    ? `${PMM_NEW_NAV_PATH}/qan`
    : `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`;

  return (
    <Tabs value={isRealtime ? 'real-time' : 'historical'}>
      <Tab
        value="historical"
        label={Messages.tabStoredMetrics}
        component={Link}
        to={historicalTo}
        data-testid="qan-header-tabs-historical-tab"
      />
      <Tab
        value="real-time"
        label={Messages.tabRealtime}
        component={Link}
        to={`${PMM_NEW_NAV_PATH}/rta`}
        data-testid="qan-header-tabs-real-time-tab"
      />
    </Tabs>
  );
};

export default QanHeaderTabs;
