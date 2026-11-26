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

import { Link } from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';
import { Fragment, type ReactNode } from 'react';
import { OM_ROUTE_NODES } from '../constants';

/** Escapes a node name so it can be matched literally inside a regular expression. */
const escapeForRegExp = (value: string) =>
  value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

/**
 * An error message with every node it names turned into a link to that node.
 *
 * A blocked node's reason has to be followable to the scan that
 * produced it, and that scan is a row on the Nodes page. A gRPC error is a plain
 * string and cannot carry a link, so the names are matched here instead.
 *
 * **Matched against the nodes we know about, not parsed out of the message.** The
 * alternative was for the backend to emit a delimiter the UI splits on, which couples
 * the two to a format and breaks silently the first time an error is reworded. Asking
 * "does this message mention a node I have?" survives any wording.
 *
 * Longest name first, so `db-1` inside `db-11` does not match the shorter one and
 * leave a stray `1` behind.
 */
export const NodeNamesLinked = ({
  text,
  nodeNames,
  omBase,
}: {
  text: string;
  nodeNames: string[];
  omBase: string;
}): ReactNode => {
  const named = nodeNames.filter((name) => name !== '' && text.includes(name));
  if (named.length === 0) {
    return text;
  }
  const pattern = new RegExp(
    `(${[...named]
      .sort((a, b) => b.length - a.length)
      .map(escapeForRegExp)
      .join('|')})`,
    'g'
  );
  // `split` with one capturing group alternates literal, match, literal, ... so the
  // odd indices are the names. Deliberately not a reduce over indexOf: a name can
  // appear more than once in one message, and every occurrence should link.
  return text.split(pattern).map((part, index) =>
    index % 2 === 1 ? (
      <Link
        key={`${part}-${index}`}
        component={RouterLink}
        to={`${omBase}/${OM_ROUTE_NODES}?node=${encodeURIComponent(part)}`}
      >
        {part}
      </Link>
    ) : (
      <Fragment key={`text-${index}`}>{part}</Fragment>
    )
  );
};
