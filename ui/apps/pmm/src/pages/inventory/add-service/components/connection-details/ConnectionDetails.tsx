import type { FC } from 'react';
import { PasswordField, TextField } from '../fields';
import { FieldGrid } from '../field-grid';
import { FormSection } from '../form-section';
import { NodesAgents } from '../nodes-agents';
import { Messages } from './ConnectionDetails.messages';
import type { ConnectionDetailsProps } from './ConnectionDetails.types';

const { labels, placeholders, tooltips } = Messages;

export const ConnectionDetails: FC<ConnectionDetailsProps> = ({
  withNodes,
  isAddressDisabled,
  children,
}) => (
  <FormSection title={Messages.title}>
    <FieldGrid>
      <TextField
        name="serviceName"
        label={labels.serviceName}
        tooltip={tooltips.serviceName}
        placeholder={placeholders.serviceName}
      />
    </FieldGrid>
    {withNodes && <NodesAgents />}
    <FieldGrid>
      <TextField
        name="address"
        label={labels.address}
        tooltip={tooltips.address}
        disabled={isAddressDisabled}
      />
      <TextField name="port" label={labels.port} tooltip={tooltips.port} />
      <TextField
        name="username"
        label={labels.username}
        tooltip={tooltips.username}
      />
      <PasswordField
        name="password"
        label={labels.password}
        tooltip={tooltips.password}
      />
      <TextField
        name="connectionTimeout"
        label={labels.timeout}
        tooltip={tooltips.timeout}
      />
      {children}
    </FieldGrid>
  </FormSection>
);
