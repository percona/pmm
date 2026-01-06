import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { Messages } from '../QanHeader.messages';
import { FC } from 'react';
import { Link } from 'react-router-dom';
import { useIsRealTimeQan } from 'hooks/utils/useLocation';
import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';

const QanHeaderTabs: FC = () => {
  const isRealTime = useIsRealTimeQan();

  return (
    <Tabs value={isRealTime ? 'real-time' : 'historical'}>
      <Tab
        value="historical"
        label={Messages.tabHistorical}
        component={Link}
        to={`${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`}
        data-testid="qan-header-tabs-historical-tab"
      />
      <Tab
        value="real-time"
        label={Messages.tabRealTime}
        component={Link}
        to={`${PMM_NEW_NAV_PATH}/rta`}
        data-testid="qan-header-tabs-real-time-tab"
      />
    </Tabs>
  );
};

export default QanHeaderTabs;
