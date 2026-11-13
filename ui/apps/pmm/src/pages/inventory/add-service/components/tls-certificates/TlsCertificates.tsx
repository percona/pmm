import type { FC } from 'react';
import { TextField } from '../fields';
import { Messages } from './TlsCertificates.messages';

const { labels, tooltips } = Messages;

export const TlsCertificates: FC = () => (
  <>
    <TextField
      name="tlsCa"
      kind="textarea"
      label={labels.tlsCa}
      tooltip={tooltips.tlsCa}
    />
    <TextField
      name="tlsKey"
      kind="textarea"
      label={labels.tlsKey}
      tooltip={tooltips.tlsKey}
    />
    <TextField
      name="tlsCert"
      kind="textarea"
      label={labels.tlsCert}
      tooltip={tooltips.tlsCert}
    />
  </>
);
