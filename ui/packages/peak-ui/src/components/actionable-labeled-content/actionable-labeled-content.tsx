import { Button, ButtonProps, Chip } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import LabeledContent, { LabeledContentProps } from '../labeled-content';

export type ActionableLabeledContentProps = LabeledContentProps & {
  techPreview?: boolean;
  actionButtonProps?: {
    dataTestId?: string;
    onClick: () => void;
    buttonText?: string;
  } & ButtonProps;
};

export const ActionableLabeledContent = ({
  label,
  techPreview,
  content,
  actionButtonProps,
  verticalStackSx,
  horizontalStackSx,
  ...rest
}: ActionableLabeledContentProps) => {
  const { dataTestId, buttonText, ...buttonProps } = actionButtonProps || {};

  return (
    <LabeledContent
      label={label}
      caption={content}
      verticalStackSx={{
        '.MuiTextField-root': {
          mt: actionButtonProps ? 0 : 1.5,
        },
        ...verticalStackSx,
      }}
      horizontalStackSx={{
        marginBottom: actionButtonProps ? 1 : 0.5,
        ...horizontalStackSx,
      }}
      horizontalStackChildrenSlot={
        <>
          {techPreview && (
            <Chip size="small" label="Technical preview" sx={{ ml: 1 }} />
          )}
          {actionButtonProps && (
            <Button
              variant="text"
              size="small"
              startIcon={<AddIcon />}
              sx={{
                width: 'fit-content',
                ml: 'auto',
              }}
              data-testid={dataTestId || 'labeled-content-action-button'}
              {...buttonProps}
            >
              {buttonText || 'Add new'}
            </Button>
          )}
        </>
      }
      {...rest}
    />
  );
};

export default ActionableLabeledContent;
