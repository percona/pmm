import { FC, useEffect } from 'react';

const PMMPage: FC = () => {
  useEffect(() => {
    console.log('pmm-page-mount');

    return console.log('pmm-page-unmount');
  }, []);
  return 'Hello from PMM';
};

export default PMMPage;
