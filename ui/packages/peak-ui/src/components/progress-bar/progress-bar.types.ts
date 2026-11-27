export type ProgressBarProps = {
  /** Test id of the bar; the label gets it with a `-label` suffix. */
  dataTestId?: string;
  /** Amount already used, drawn as the solid part from the start of the bar; hidden when `buffer` exceeds `total`. */
  value: number;
  /** Used plus reserved amount, drawn from the start of the bar behind `value`; above `total`, the whole bar turns warning-colored. */
  buffer: number;
  /** Amount that fills the bar completely. */
  total: number;
  /** Text shown above the bar, aligned right, such as "12 of 20 GB". */
  label: string;
};
