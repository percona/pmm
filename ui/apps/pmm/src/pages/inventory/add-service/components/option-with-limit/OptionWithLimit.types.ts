export interface OptionWithLimitProps {
  name: string;
  label: string;
  options: { value: string; label: string }[];
  // The option that lets the user type the limit.
  editableOption: string;
  limitName: string;
  limitLabel: string;
  limitTooltip?: string;
}
