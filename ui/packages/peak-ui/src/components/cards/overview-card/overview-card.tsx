import { ReactNode } from 'react';
import {
  Card as MuiCard,
  CardProps as MuiCardProps,
  CardContent,
  CardContentProps,
  CardHeader,
  CardHeaderProps,
} from '@mui/material';
import { DatabaseIcon } from '../../icons';

export interface OverviewCardProps extends Omit<MuiCardProps, 'content'> {
  cardHeaderProps?: CardHeaderProps;
  cardContentProps?: CardContentProps;
  dataTestId: string;
  children: ReactNode;
}

const OverviewCard = ({
  cardHeaderProps,
  children,
  sx,
  cardContentProps,
  dataTestId,
  ...props
}: OverviewCardProps) => {
  return (
    <MuiCard
      variant="grey"
      sx={{ width: '368px', height: 'fit-content', ...sx }}
      data-testid={dataTestId}
      {...props}
    >
      {cardHeaderProps?.title && (
        <CardHeader
          data-testid={`${dataTestId}-card-header`}
          title={cardHeaderProps?.title}
          avatar={<DatabaseIcon />}
          {...cardHeaderProps}
        />
      )}
      <CardContent {...cardContentProps}>{children}</CardContent>
    </MuiCard>
  );
};

export default OverviewCard;
