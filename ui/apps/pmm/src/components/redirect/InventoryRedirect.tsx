import { getInventoryRedirectUrl } from 'pages/inventory/Inventory.utils';
import { Navigate, useLocation, useParams } from 'react-router-dom';

const InventoryRedirect = () => {
  const { '*': path = '' } = useParams();
  const { search } = useLocation();

  return <Navigate to={getInventoryRedirectUrl(path, search)} replace />;
};

export default InventoryRedirect;
