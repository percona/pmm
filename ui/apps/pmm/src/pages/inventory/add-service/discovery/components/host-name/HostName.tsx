import { Fragment, type FC } from 'react';
import type { HostNameProps } from './HostName.types';

// Wraps a long host name after a dot, and inside a label only when the label alone is too wide.
// One span, since table cells lay their children out as flex items.
export const HostName: FC<HostNameProps> = ({ host }) => (
  <span>
    {host.split('.').map((label, index, labels) => (
      <Fragment key={index}>
        <span
          style={{
            display: 'inline-block',
            maxWidth: '100%',
            overflowWrap: 'anywhere',
          }}
        >
          {label}
          {index < labels.length - 1 && '.'}
        </span>
        {index < labels.length - 1 && <wbr />}
      </Fragment>
    ))}
  </span>
);
