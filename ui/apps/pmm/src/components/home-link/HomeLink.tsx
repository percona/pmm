import { Link, LinkProps } from '@mui/material';
import {
  PMM_HOME_URL,
  PMM_NEW_NAV_HOME_URL,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { useUpdates } from 'contexts/updates';
import { FC, useMemo, useState } from 'react';
import { UpdateStatus } from 'types/updates.types';
import { ClientsModal } from './clients-modal';
import { useLocation, Link as RouterLink } from 'react-router-dom';

export const HomeLink: FC<LinkProps> = ({ children, sx, ...props }) => {
  const { status, areClientsUpToDate } = useUpdates();
  const [modalOpen, setModalOpen] = useState(false);
  const location = useLocation();
  const isOnClientsPage = useMemo(
    () => location.pathname.endsWith('/updates/clients'),
    [location]
  );
  const linkProps = useMemo(() => {
    if (location.pathname.includes(PMM_NEW_NAV_PATH)) {
      return {
        to: PMM_NEW_NAV_HOME_URL,
        component: RouterLink,
      };
    }

    return {
      href: PMM_HOME_URL,
    };
  }, [location.pathname]);
  const homeLinkProps = useMemo(() => {
    if (
      status === UpdateStatus.UpdateClients &&
      !areClientsUpToDate &&
      isOnClientsPage
    ) {
      return {
        onClick: () => setModalOpen(true),
      };
    }

    return linkProps;
  }, [linkProps, status, areClientsUpToDate, isOnClientsPage]);

  return (
    <>
      <ClientsModal
        homeLinkProps={linkProps}
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
      />
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
