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

import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { NodeNamesLinked } from '../src/components/NodeNamesLinked';

const renderText = (text: string, nodeNames: string[]) =>
  render(
    <MemoryRouter>
      <NodeNamesLinked text={text} nodeNames={nodeNames} omBase="/operations" />
    </MemoryRouter>
  );

describe('NodeNamesLinked', () => {
  // P6's last clause: the reason has to be followable to the scan that produced it.
  it('links every node the message names, to that node on the Nodes page', () => {
    renderText(
      '2 of the selected node(s) cannot be installed onto -- db-01: no scan; db-03: no agent.',
      ['db-01', 'db-02', 'db-03']
    );

    expect(screen.getByRole('link', { name: 'db-01' })).toHaveAttribute(
      'href',
      '/operations/nodes?node=db-01'
    );
    expect(screen.getByRole('link', { name: 'db-03' })).toHaveAttribute(
      'href',
      '/operations/nodes?node=db-03'
    );
    // A node the message does not mention is not linked into it.
    expect(screen.queryByRole('link', { name: 'db-02' })).toBeNull();
  });

  // The message is still readable as a sentence, not reduced to a list of links.
  it('keeps the surrounding text', () => {
    renderText('db-01: no scan has reported its operating system', ['db-01']);

    expect(
      screen.getByText(/no scan has reported its operating system/)
    ).toBeInTheDocument();
  });

  // Longest first. Matching 'db-1' inside 'db-11' would link the wrong node and
  // leave a stray '1' outside the link.
  it('prefers the longer name when one is a prefix of another', () => {
    renderText('db-11 is blocked', ['db-1', 'db-11']);

    expect(screen.getByRole('link', { name: 'db-11' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'db-1' })).toBeNull();
  });

  it('links every occurrence of a name, not only the first', () => {
    renderText('db-01 failed, and db-01 has no agent', ['db-01']);

    expect(screen.getAllByRole('link', { name: 'db-01' })).toHaveLength(2);
  });

  // A message about nothing we know about is left exactly as the backend wrote it,
  // rather than rendered through a code path that could mangle it.
  it('renders an unrecognised message unchanged', () => {
    renderText('PMM Extensions is not configured', ['db-01']);

    expect(
      screen.getByText('PMM Extensions is not configured')
    ).toBeInTheDocument();
    expect(screen.queryByRole('link')).toBeNull();
  });

  // Node names are not regular expressions. A name with a metacharacter in it must
  // match literally rather than blowing up or matching the wrong span.
  it('treats a name containing regex metacharacters literally', () => {
    renderText('db.01 is blocked', ['db.01']);

    expect(screen.getByRole('link', { name: 'db.01' })).toHaveAttribute(
      'href',
      '/operations/nodes?node=db.01'
    );
  });
});
