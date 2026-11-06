import { PMM_DUMP_PATH } from 'lib/constants';
import { Navigate, useParams } from 'react-router-dom';

const PmmDumpRedirect = () => {
  const { '*': path = '' } = useParams();
  return (
    <Navigate to={path ? `${PMM_DUMP_PATH}/${path}` : PMM_DUMP_PATH} replace />
  );
};

export default PmmDumpRedirect;
