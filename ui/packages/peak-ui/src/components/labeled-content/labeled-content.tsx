import { Stack, Typography } from '@mui/material';
import { LabeledContentProps } from './labeled-content.types';

const LabeledContent = ({
  label,
  caption,
  children,
  isRequired = false,
  verticalStackSx: verticalSx = {
    '.MuiTextField-root': {},
    '.MuiAutocomplete-root': {},
  },
  horizontalStackSx: horizontalStackSx,
  horizontalStackChildrenSlot,
  ...typographyProps
}: LabeledContentProps) => {
  const {
    // @ts-expect-error destructuring CSS selector keys that TypeScript doesn't recognize as valid object keys
    '.MuiTextField-root': textFieldRootSx,
    // @ts-expect-error destructuring CSS selector keys that TypeScript doesn't recognize as valid object keys
    '.MuiAutocomplete-root': autocompleteRootSx,
    ...verticalStackSx
  } = verticalSx;
  return (
    <Stack
      sx={{
        '.MuiTextField-root': {
          mt: 1.5,
          ...textFieldRootSx,
        },
        '.MuiAutocomplete-root': {
          mt: 1.5,
          ...autocompleteRootSx,
        },
        mt: 2,
        ...verticalStackSx,
      }}
    >
      <Stack
        sx={{
          flexDirection: 'row',
          alignItems: 'center',
          marginBottom: 0.5,
          ...horizontalStackSx,
        }}
      >
        <Typography variant="sectionHeading" {...typographyProps}>
          {label}
          {isRequired && <span>*</span>}
        </Typography>
        {horizontalStackChildrenSlot}
      </Stack>
      {caption && <Typography variant="body2">{caption}</Typography>}
      {children}
    </Stack>
  );
};

export default LabeledContent;
