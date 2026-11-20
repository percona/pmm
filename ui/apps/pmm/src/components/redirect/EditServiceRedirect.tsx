import { getEditServiceUrl } from 'pages/inventory/Inventory.utils';
import { Navigate, useParams } from 'react-router-dom';

const EditServiceRedirect = () => {
  const { serviceId = '' } = useParams();

  return <Navigate to={getEditServiceUrl(serviceId)} replace />;
};

export default EditServiceRedirect;
