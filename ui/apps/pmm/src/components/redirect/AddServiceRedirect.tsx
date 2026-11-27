import { PMM_ADD_SERVICE_PATH } from 'lib/constants';
import {
  getAddServicePath,
  isAddServiceType,
} from 'pages/inventory/add-service/AddServicePage.utils';
import { Navigate, useParams } from 'react-router-dom';

const AddServiceRedirect = () => {
  const { type } = useParams<{ type: string }>();
  return (
    <Navigate
      to={
        isAddServiceType(type) ? getAddServicePath(type) : PMM_ADD_SERVICE_PATH
      }
      replace
    />
  );
};

export default AddServiceRedirect;
