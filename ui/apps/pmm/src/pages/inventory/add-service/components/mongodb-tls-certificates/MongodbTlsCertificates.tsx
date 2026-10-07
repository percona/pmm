import type { FC } from 'react';
import { PasswordField, TextField } from '../fields';
import { Messages as TlsMessages } from '../tls-certificates/TlsCertificates.messages';
import { Messages } from './MongodbTlsCertificates.messages';

const { labels, tooltips } = TlsMessages;

export const MongodbTlsCertificates: FC = () => (
  <>
    <PasswordField
      name="tlsCertificateKeyFilePassword"
      label={Messages.password}
    />
    <TextField
      name="tlsCertificateKey"
      kind="textarea"
      label={labels.tlsKey}
      tooltip={tooltips.tlsKey}
    />
    <TextField
      name="tlsCa"
      kind="textarea"
      label={labels.tlsCa}
      tooltip={tooltips.tlsCa}
    />
  </>
);
