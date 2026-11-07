import type { FC } from 'react';
import { TextField } from '../../../components/fields';
import { Messages } from '../PostgreSQLForm.messages';

export const DatabaseField: FC = () => (
  <TextField
    name="database"
    label={Messages.database.label}
    placeholder={Messages.database.placeholder}
    tooltip={Messages.database.tooltip}
  />
);
