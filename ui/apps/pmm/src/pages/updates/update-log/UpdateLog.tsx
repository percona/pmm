import { Button, Stack } from '@mui/material';
import { Modal } from 'components/modal';
import { FC, useCallback, useEffect, useRef, useState } from 'react';
import { UpdateLogContent } from './update-log-content';
import { useUpdateLog } from './UpdateLog.hooks';
import { UpdateLogProps } from './UpdateLog.types';
import { useUpdates } from 'contexts/updates';
import { UpdateStatus } from 'types/updates.types';
import { hardReloadPage } from 'utils/dom.utils';
import { Messages } from './UpdateLog.messages';

export const UpdateLog: FC<UpdateLogProps> = ({
  authToken,
  upgradeVersion,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const { output, isDone } = useUpdateLog(authToken);
  const { setStatus } = useUpdates();
  const reloadedRef = useRef(false);

  const handleOpen = useCallback(() => {
    setIsOpen(true);
  }, []);

  const handleClose = useCallback(() => {
    setIsOpen(false);
  }, []);

  useEffect(() => {
    if (!isDone || reloadedRef.current) {
      return;
    }

    reloadedRef.current = true;
    setStatus(UpdateStatus.Completed);
    hardReloadPage(upgradeVersion);
  }, [isDone, setStatus, upgradeVersion]);

  return (
    <>
      <Button variant="text" onClick={handleOpen}>
        {Messages.checkLog}
      </Button>
      <Modal title={Messages.modalTitle} open={isOpen} onClose={handleClose}>
        <Stack>
          <UpdateLogContent content={output} />
          <Stack direction="row" justifyContent="end" sx={{ pt: 2 }}>
            <Button variant="text" onClick={handleClose}>
              {Messages.modalClose}
            </Button>
          </Stack>
        </Stack>
      </Modal>
    </>
  );
};
