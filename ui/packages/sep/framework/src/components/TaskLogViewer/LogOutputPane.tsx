/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { LazyLog } from '@melloware/react-logviewer';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useLayoutEffect, useRef, useState } from 'react';

export interface LogOutputPaneProps {
  text: string;
  wrap: boolean;
  enableSearch?: boolean;
  height?: number | string;
  emptyLabel?: string;
}

export function LogOutputPane({
  text,
  wrap,
  enableSearch = true,
  height = 400,
  emptyLabel = 'No output yet.',
}: LogOutputPaneProps) {
  if (!text) {
    return (
      <Box sx={{ height, width: '100%', p: 2, color: 'text.secondary' }}>
        <Typography variant="body2">{emptyLabel}</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ height, width: '100%' }}>
      <AppendingLog text={text} wrap={wrap} enableSearch={enableSearch} />
    </Box>
  );
}

interface AppendedText {
  log: LazyLog | null;
  text: string;
  endsMidLine: boolean;
}

/**
 * Hands `text` to the log by appending what is new rather than replacing it.
 *
 * LazyLog clears and re-parses its whole buffer whenever its `text` prop
 * changes, and that cleared list paints (blank, then scrolled to the first
 * line) before `follow` brings it back to the end, so a streaming log
 * flickered on every chunk. Its external mode keeps the lines already shown.
 * Every append ends the log's last line, so text that stops mid-line is shown
 * as a whole line, and if more of that line arrives the log is rebuilt from
 * the full text.
 */
function AppendingLog({
  text,
  wrap,
  enableSearch,
}: Pick<LogOutputPaneProps, 'text' | 'wrap' | 'enableSearch'>) {
  const logRef = useRef<LazyLog>(null);
  const appendedRef = useRef<AppendedText>({
    log: null,
    text: '',
    endsMidLine: false,
  });
  const [generation, setGeneration] = useState(0);

  useLayoutEffect(() => {
    const log = logRef.current;
    if (!log) {
      return;
    }
    const appended =
      appendedRef.current.log === log
        ? appendedRef.current
        : { log, text: '', endsMidLine: false };
    if (!text.startsWith(appended.text)) {
      setGeneration((value) => value + 1);
      return;
    }
    let added = text.slice(appended.text.length);
    let { endsMidLine } = appended;
    if (endsMidLine && added) {
      if (!added.startsWith('\n')) {
        setGeneration((value) => value + 1);
        return;
      }
      added = added.slice(1);
      endsMidLine = false;
    }
    if (added) {
      log.appendLines([added]);
      endsMidLine = !added.endsWith('\n');
    }
    appendedRef.current = { log, text, endsMidLine };
  }, [text, generation]);

  return (
    <LazyLog
      key={generation}
      ref={logRef}
      external
      enableSearch={enableSearch}
      wrapLines={wrap}
      follow
      selectableLines
      caseInsensitive
    />
  );
}
