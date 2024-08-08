import { Link, LinkProps } from '@mui/material';
import { PMM_HOME_URL } from 'constants';
import { useUpdates } from 'contexts/updates';
import { FC, useState } from 'react';
import { UpdateStatus } from 'types/updates.types';
import { ClientsModal } from './clients-modal';

export const HomeLink: FC<LinkProps> = ({ children, sx, ...props }) => {
  const { status } = useUpdates();
  const [modalOpen, setModalOpen] = useState(false);
  const homeLinkProps =
    status === UpdateStatus.UpdateClients
      ? {
          onClick: () => setModalOpen(true),
        }
      : {
          href: PMM_HOME_URL,
        };

  return (
    <>
      <ClientsModal isOpen={modalOpen} onClose={() => setModalOpen(false)} />
      <Link
        {...props}
        sx={[{ cursor: 'pointer ' }, ...(Array.isArray(sx) ? sx : [sx])]}
        {...homeLinkProps}
      >
        {children}
      </Link>
    </>
  );
};
