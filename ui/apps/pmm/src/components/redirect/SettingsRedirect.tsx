import { Navigate, useParams } from 'react-router-dom';

const SettingsPageRedirect = () => {
  const { tab } = useParams();
  return <Navigate to={`/settings/${tab}`} replace />;
};

export default SettingsPageRedirect;
