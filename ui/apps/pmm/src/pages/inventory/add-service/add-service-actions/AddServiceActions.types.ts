// Cancel either links to `cancelTo` or is handled in place by `onCancel`.
type CancelProps =
  | { cancelTo: string; onCancel?: never }
  | { onCancel: () => void; cancelTo?: never };

export type AddServiceActionsProps = CancelProps & {
  // Renders a submit button for the surrounding form.
  submitLabel?: string;
  isSubmitting?: boolean;
  // Keeps the actions in view at the bottom of a long form.
  isSticky?: boolean;
};
