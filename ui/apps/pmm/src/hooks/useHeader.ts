import QanHeader from 'components/main/header/QanHeader';
import { useLocation } from 'react-router-dom';

const useHeader = () => {
  const pathname = useLocation().pathname;
  const isQan = pathname.includes('pmm-qan') || pathname.includes('rta');
  const Component = isQan ? QanHeader : null;
  const isCustomView = isQan;

  return { visible: isCustomView, Component: Component };
};

export default useHeader;
