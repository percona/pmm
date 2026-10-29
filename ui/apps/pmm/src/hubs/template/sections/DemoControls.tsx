import Box from '@mui/material/Box';
import Divider from '@mui/material/Divider';
import FormControlLabel from '@mui/material/FormControlLabel';
import Switch from '@mui/material/Switch';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Typography from '@mui/material/Typography';
import { type FC, useId } from 'react';
import { PrototypeControls } from 'components/prototype-controls';
import { Messages } from '../hub.messages';
import { DEFAULT_HUB_PARAMS } from '../hub.params';
import type { HubParams } from '../hub.types';

const SCENARIOS = ['healthy', 'incomplete', 'unhealthy'] as const;

export interface DemoControlsProps {
  /** The technology's name, as in "Operations for MongoDB". */
  name: string;
  params: HubParams;
  setParams: (next: Partial<HubParams>) => void;
  /** Whether the page shows Operations, after the switches. */
  operations: boolean;
  /** Only an admin can have Operations on. */
  admin: boolean;
}

/** Prototype only: the hub's demo states in the shared prototype controls, all kept in the URL. */
export const DemoControls: FC<DemoControlsProps> = ({
  name,
  params,
  setParams,
  operations,
  admin,
}) => {
  const messages = Messages.demo;
  const scenarioLabelId = useId();
  // Label left, switch right; the switch's own padding reaches into the panel's, as its track lines up with the edge.
  // The row keeps only the label's height, so the 38px switch doesn't add its padding to the space around the rules.
  const switchRow = {
    ml: 0,
    mr: -1.5,
    my: -1,
    gap: 2,
    justifyContent: 'space-between',
  };

  return (
    <PrototypeControls title={messages.trigger}>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {/* As strong as the switches' labels, which name settings the same way. */}
        <Typography id={scenarioLabelId}>{messages.scenarioLabel}</Typography>
        <ToggleButtonGroup
          exclusive
          size="small"
          value={
            (SCENARIOS as readonly string[]).includes(params.scenario)
              ? params.scenario
              : null
          }
          onChange={(_, scenario: (typeof SCENARIOS)[number] | null) =>
            scenario !== null && setParams({ scenario })
          }
          aria-labelledby={scenarioLabelId}
          sx={{ '& .MuiToggleButton-root': { whiteSpace: 'nowrap' } }}
        >
          {SCENARIOS.map((scenario) => (
            <ToggleButton key={scenario} value={scenario}>
              {messages.scenario[scenario]}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
      </Box>
      <Divider />
      <FormControlLabel
        labelPlacement="start"
        label={messages.stale}
        sx={switchRow}
        control={
          <Switch
            checked={params.freshness === 'stale'}
            // Off drops the simulation and returns to the real 30s refresh.
            onChange={(_, stale) =>
              setParams(
                stale
                  ? { freshness: 'stale' }
                  : {
                      freshness: 'live',
                      refresh: DEFAULT_HUB_PARAMS.refresh,
                    }
              )
            }
          />
        }
      />
      <Divider />
      <FormControlLabel
        labelPlacement="start"
        label={messages.operations(name)}
        disabled={!admin}
        title={admin ? undefined : messages.adminOnly}
        sx={switchRow}
        control={
          <Switch
            checked={operations}
            onChange={(_, on) => setParams({ ops: on ? 'on' : 'off' })}
          />
        }
      />
    </PrototypeControls>
  );
};
