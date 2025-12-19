import { SkeletonProps } from '@mui/material';

export type LoadableChildrenProps = {
  children: React.ReactNode;
  loading?: boolean;
  skeletonProps?: SkeletonProps;
};
