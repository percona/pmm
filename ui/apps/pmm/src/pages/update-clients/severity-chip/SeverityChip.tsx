import { FC, useMemo } from 'react';
import { SeverityChipProps } from './SeverityChip.types';
import { Chip, ChipOwnProps } from '@mui/material';
import { AgentUpdateSeverity } from 'types/agent.types';

import WarningIcon from '@mui/icons-material/Warning';
import CheckIcon from '@mui/icons-material/Check';
import { Messages } from '../UpdateClients.messages';

export const SeverityChip: FC<SeverityChipProps> = ({ severity }) => {
  const label = useMemo<string>(() => {
    if (severity === AgentUpdateSeverity.CRITICAL) {
      return Messages.severity.critical;
    } else if (severity === AgentUpdateSeverity.REQUIRED) {
      return Messages.severity.required;
    } else if (severity === AgentUpdateSeverity.UP_TO_DATE) {
      return Messages.severity.upToDate;
    } else if (severity === AgentUpdateSeverity.UNSUPPORTED) {
      return Messages.severity.unsupported;
    }

    return Messages.severity.unspecified;
  }, [severity]);
  const color = useMemo<ChipOwnProps['color']>(() => {
    if (severity === AgentUpdateSeverity.CRITICAL) {
      return 'error';
    } else if (severity === AgentUpdateSeverity.REQUIRED) {
      return 'warning';
    } else if (severity === AgentUpdateSeverity.UP_TO_DATE) {
      return 'success';
    }

    return 'info';
  }, [severity]);
  const icon = useMemo<ChipOwnProps['icon'] | undefined>(() => {
    if (severity === AgentUpdateSeverity.CRITICAL) {
      return <WarningIcon />;
    } else if (severity === AgentUpdateSeverity.UP_TO_DATE) {
      return <CheckIcon />;
    }

    return undefined;
  }, [severity]);

  return <Chip icon={icon} label={label} variant="filled" color={color} />;
};
