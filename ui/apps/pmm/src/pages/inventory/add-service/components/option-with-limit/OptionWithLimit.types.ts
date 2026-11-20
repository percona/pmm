export interface OptionWithLimitProps {
  name: string;
  label: string;
  options: { value: string; label: string }[];
  // The option that lets the user type the limit.
  editableOption: string;
  limitName: string;
  // What each other option sends, shown in place of the editable limit.
  presets?: Partial<Record<string, number>>;
  limitLabel: string;
  limitTooltip?: string;
}
