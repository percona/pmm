import React from 'react';
import { Skeleton } from '@mui/material';
import { LoadableChildrenProps } from './loadable-children.types';

const LoadableChildren = ({
  children,
  loading,
  skeletonProps,
}: LoadableChildrenProps) => (
  <>
    {React.Children.map(children, (child) =>
      loading ? <Skeleton {...skeletonProps} /> : <>{child}</>
    )}
  </>
);

export default LoadableChildren;
