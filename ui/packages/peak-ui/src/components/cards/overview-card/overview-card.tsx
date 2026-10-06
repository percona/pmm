import { ReactNode } from 'react';
import {
  Card as MuiCard,
  CardProps as MuiCardProps,
  CardContent,
  CardContentProps,
  CardHeader,
  CardHeaderProps,
} from '@mui/material';

export interface OverviewCardProps extends Omit<MuiCardProps, 'content'> {
  /** MUI CardHeader props; the header only renders when `title` is set here. */
  cardHeaderProps?: CardHeaderProps;
  /** MUI CardContent props for the padded inner area. */
  cardContentProps?: CardContentProps;
  /** Test id of the card; the header gets it as a prefix. */
  dataTestId: string;
  /** Free-form content of the card. */
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
          {...cardHeaderProps}
        />
      )}
      <CardContent {...cardContentProps}>{children}</CardContent>
    </MuiCard>
  );
};

export default OverviewCard;
