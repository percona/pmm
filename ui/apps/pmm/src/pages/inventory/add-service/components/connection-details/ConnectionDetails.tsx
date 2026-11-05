import type { FC } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import type { TlsFormValues } from '../../service-form/ServiceForm.types';
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
  requiresCredentials,
  children,
}) => {
  const { control } = useFormContext<TlsFormValues>();
  const tls = useWatch({ control, name: 'tls' });
  const credentialsRequired = requiresCredentials && !tls;

  return (
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
          required
        />
        <TextField
          name="port"
          label={labels.port}
          tooltip={tooltips.port}
          required
        />
        <TextField
          name="username"
          label={labels.username}
          tooltip={tooltips.username}
          required={credentialsRequired}
        />
        <PasswordField
          name="password"
          label={labels.password}
          tooltip={tooltips.password}
          required={credentialsRequired}
        />
        <TextField
          name="connectionTimeout"
          label={labels.timeout}
          tooltip={tooltips.timeout}
          placeholder={placeholders.timeout}
        />
        {children}
      </FieldGrid>
    </FormSection>
  );
};
