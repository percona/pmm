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

/**
 * How the text handed to the log so far ends: on a newline, on a carriage
 * return whose pairing newline may still arrive, or partway through a line.
 */
type LastLineEnd = 'closed' | 'afterCarriageReturn' | 'open';

interface AppendedText {
  log: LazyLog | null;
  text: string;
  lastLine: LastLineEnd;
}

function lastLineEnd(text: string): LastLineEnd {
  if (text.endsWith('\n')) {
    return 'closed';
  }
  return text.endsWith('\r') ? 'afterCarriageReturn' : 'open';
}

/**
 * Hands `text` to the log by appending what is new rather than replacing it.
 *
 * LazyLog clears and re-parses its whole buffer whenever its `text` prop
 * changes, and that cleared list paints (blank, then scrolled to the first
 * line) before `follow` brings it back to the end, so a streaming log
 * flickered on every chunk. Its external mode keeps the lines already shown.
 *
 * Every append ends the log's last line with a newline, which LazyLog pairs
 * with a trailing carriage return, the other line end it recognises. So a
 * newline arriving after a carriage return, or a line end arriving after an
 * open line, is already accounted for and is dropped. More of an open line
 * cannot be appended, so the log is rebuilt from the full text instead.
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
    lastLine: 'closed',
  });
  const [generation, setGeneration] = useState(0);

  useLayoutEffect(() => {
    const log = logRef.current;
    if (!log) {
      return;
    }
    const appended: AppendedText =
      appendedRef.current.log === log
        ? appendedRef.current
        : { log, text: '', lastLine: 'closed' };
    if (!text.startsWith(appended.text)) {
      setGeneration((value) => value + 1);
      return;
    }
    let added = text.slice(appended.text.length);
    let { lastLine } = appended;
    if (lastLine === 'open' && added.startsWith('\r')) {
      added = added.slice(1);
      lastLine = 'afterCarriageReturn';
    }
    if (lastLine !== 'closed' && added.startsWith('\n')) {
      added = added.slice(1);
      lastLine = 'closed';
    } else if (lastLine === 'open' && added) {
      setGeneration((value) => value + 1);
      return;
    }
    if (added) {
      log.appendLines([added]);
      lastLine = lastLineEnd(added);
    }
    appendedRef.current = { log, text, lastLine };
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
