import type { FC } from 'react';
import { TextField } from '../../../components/fields';
import { Messages } from '../RdsServiceForm.messages';

export const InstanceIdField: FC = () => (
  <TextField
    name="instanceId"
    label={Messages.instanceId.label}
    placeholder={Messages.instanceId.placeholder}
    tooltip={Messages.instanceId.tooltip}
  />
);
