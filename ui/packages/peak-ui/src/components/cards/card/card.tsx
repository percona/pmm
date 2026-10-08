import { ReactNode } from 'react';
import {
  Card as MuiCard,
  Typography,
  CardProps as MuiCardProps,
  CardContent,
  Box,
  Button,
  CardActions,
  ButtonProps,
  TypographyProps,
  CardContentProps,
  BoxProps,
  CardActionsProps,
} from '@mui/material';
import { kebabize } from '@/utils';

export interface CardProps extends Omit<MuiCardProps, 'content'> {
  /** Main body of the card, where the contents of the card go. */
  content: ReactNode;
  /** Prefix for the test ids of the card, its title, body, and buttons row; each button's test id comes from its `text`. */
  dataTestId: string;
  /** Buttons shown at the bottom of the card, after its body. Each button takes MUI Button props plus its `text`. */
  cardActions?: ActionProps[];
  /** MUI CardActions props for the buttons row, such as alignment. */
  cardActionsProps?: CardActionsProps;
  /** MUI Typography props for the title, such as variant or sx. */
  headerProps?: TypographyProps;
  /** MUI CardContent props for the padded inner area. */
  cardContentProps?: CardContentProps;
  /** MUI Box props for the wrapper around `content`, such as layout styles. */
  contentWrapperProps?: BoxProps;
}

export interface ActionProps extends ButtonProps {
  /** Label of the button; also drives its test id. */
  text: string;
}

const Card = ({
  title,
  content,
  sx,
  cardActions,
  headerProps,
  cardContentProps,
  contentWrapperProps,
  cardActionsProps,
  dataTestId,
  ...props
}: CardProps) => {
  return (
    <MuiCard
      data-testid={`${dataTestId}-card`}
      sx={{ width: '320px', height: 'fit-content', ...sx }}
      {...props}
    >
      <CardContent
        data-testid={`${dataTestId}-card-content`}
        {...cardContentProps}
        sx={{
          '&:last-child': {
            p: 2,
          },
          ...cardContentProps?.sx,
        }}
      >
        {title && (
          <Typography
            data-testid={`${dataTestId}-card-header`}
            variant="h5"
            {...headerProps}
            sx={{ mb: 4, ...headerProps?.sx }}
          >
            {title}
          </Typography>
        )}
        <Box data-testid={`${dataTestId}-card-content-wrapper`} {...contentWrapperProps}>
          {content}
        </Box>
        {cardActions && (
          <CardActions
            data-testid={`${dataTestId}-card-actions`}
            {...cardActionsProps}
            sx={{ p: 0, mt: 4, ...cardActionsProps?.sx }}
          >
            {cardActions.map(({ text, ...buttonProps }) => (
              <Button key={text} data-testid={`${kebabize(text)}-button`} {...buttonProps}>
                {text}
              </Button>
            ))}
          </CardActions>
        )}
      </CardContent>
    </MuiCard>
  );
};

export default Card;
