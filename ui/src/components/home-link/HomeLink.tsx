import { Link, LinkProps } from '@mui/material';
import { PMM_HOME_URL } from 'constants';
import { useUpdates } from 'contexts/updates';
import { FC, useMemo, useState } from 'react';
import { UpdateStatus } from 'types/updates.types';
import { ClientsModal } from './clients-modal';
import { useLocation } from 'react-router-dom';

export const HomeLink: FC<LinkProps> = ({ children, sx, ...props }) => {
  const { status, areClientsUpToDate } = useUpdates();
  const [modalOpen, setModalOpen] = useState(false);
  const location = useLocation();
  const isOnClientsPage = useMemo(
    () => location.pathname.startsWith('/updates/clients'),
    [location]
  );
  const homeLinkProps =
    (status === UpdateStatus.UpdateClients || !areClientsUpToDate) &&
    isOnClientsPage
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
