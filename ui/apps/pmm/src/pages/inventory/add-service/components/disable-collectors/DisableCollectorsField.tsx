import type { FC } from 'react';
import { TextField } from '../fields';
import { Messages } from './DisableCollectorsField.messages';

export const DisableCollectorsField: FC = () => (
  <TextField
    name="disableCollectors"
    label={Messages.label}
    placeholder={Messages.placeholder}
    helperText={Messages.description}
  />
);
