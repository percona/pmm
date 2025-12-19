import { ReactNode } from "react";
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
} from "@mui/material";
import { kebabize } from "@/utils";

export interface CardProps extends Omit<MuiCardProps, "content"> {
  content: ReactNode;
  dataTestId: string;
  cardActions?: ActionProps[];
  cardActionsProps?: CardActionsProps;
  headerProps?: TypographyProps;
  cardContentProps?: CardContentProps;
  contentWrapperProps?: BoxProps;
}

export interface ActionProps extends ButtonProps {
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
      sx={{ width: "320px", height: "fit-content", ...sx }}
      {...props}
    >
      <CardContent
        data-testid={`${dataTestId}-card-content`}
        {...cardContentProps}
        sx={{
          "&:last-child": {
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
        <Box
          data-testid={`${dataTestId}-card-content-wrapper`}
          {...contentWrapperProps}
        >
          {content}
        </Box>
        {cardActions && (
          <CardActions
            data-testid={`${dataTestId}-card-actions`}
            {...cardActionsProps}
            sx={{ p: 0, mt: 4, ...cardActionsProps?.sx }}
          >
            {cardActions.map(({ text, ...buttonProps }) => (
              <Button
                key={text}
                data-testid={`${kebabize(text)}-button`}
                {...buttonProps}
              >
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
