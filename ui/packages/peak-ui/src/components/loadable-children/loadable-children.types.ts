import { SkeletonProps } from '@mui/material';

export type LoadableChildrenProps = {
  /** Content to show once loaded; each child is replaced by its own skeleton while loading. */
  children: React.ReactNode;
  /** Shows skeleton placeholders instead of the children. */
  loading?: boolean;
  /** MUI Skeleton props such as width, height, or variant, to match the shape of the real content. */
  skeletonProps?: SkeletonProps;
};
