import { PMM_DUMP_PATH } from 'lib/constants';
import { Navigate, useLocation, useParams } from 'react-router-dom';

const PmmDumpRedirect = () => {
  const { '*': path = '' } = useParams();
  const { search, hash } = useLocation();
  const pathname = path ? `${PMM_DUMP_PATH}/${path}` : PMM_DUMP_PATH;

  return <Navigate to={{ pathname, search, hash }} replace />;
};

export default PmmDumpRedirect;
