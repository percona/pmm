import { SxProps, Theme, TypographyProps } from '@mui/material';

export type LabeledContentProps = {
  /** Section heading shown above the content. */
  label?: string;
  /** Explanatory text shown under the heading. */
  caption?: string;
  /** Content shown on the heading row, to the right of the label, such as a button or chip. */
  horizontalStackChildrenSlot?: React.ReactNode;
  /** Adds a required asterisk after the heading. */
  isRequired?: boolean;
  /** Styles for the whole block: heading, caption, and children stacked vertically. */
  verticalStackSx?: SxProps<Theme>;
  /** Styles for the heading row. */
  horizontalStackSx?: SxProps<Theme>;
} & TypographyProps;
