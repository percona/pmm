import { Navigate, useLocation } from 'react-router-dom';

const Redirect = () => {
  const location = useLocation();
  // Remove /next prefix from the pathname
  const newPath = location.pathname.replace(/^\/next/, '') || '/';
  return <Navigate to={newPath} replace />;
};

export default Redirect;
