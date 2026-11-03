import type { FC } from 'react';
import { TextField } from '../fields';
import { Messages } from './MaxQueryLengthField.messages';

export const MaxQueryLengthField: FC = () => (
  <TextField
    name="maxQueryLength"
    kind="number"
    label={Messages.label}
    tooltip={Messages.tooltip}
  />
);
