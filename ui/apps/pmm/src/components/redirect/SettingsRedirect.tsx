import { TabValue } from 'pages/settings/Settings.types';
import { Navigate, useParams } from 'react-router-dom';

const SettingsPageRedirect = () => {
  const { tab = '' } = useParams<{ tab: TabValue }>();
  return <Navigate to={`/settings/${tab}`} replace />;
};

export default SettingsPageRedirect;
