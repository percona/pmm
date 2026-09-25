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

import { Children, type ReactElement } from 'react';
import { describe, expect, it } from 'vitest';
import { BackupMongoApp } from './BackupMongoApp';
import {
  restoreMongoCreateForm,
  restoreMongoEditForm,
} from './restoreMongoCreateForm';

type AnyProps = Record<string, unknown>;

const BASE_PATH = '/om/mongodb-backups';

// BackupMongoApp holds no hooks of its own, so calling it yields the element
// tree without mounting. Walk the <Routes> to read each tab's SchemaDrivenPlugin
// props — this fails if the restores tab loses its custom edit-form wiring,
// which prop-only slot tests cannot catch.
function schemaAppPropsByRoutePath(): Record<string, AnyProps> {
  const tree = BackupMongoApp({ basePath: BASE_PATH }) as ReactElement<{
    children: ReactElement[];
  }>;
  const children = Children.toArray(
    tree.props.children
  ) as ReactElement<AnyProps>[];
  const routesEl = children.find(
    (child) =>
      Array.isArray(child.props.children) &&
      (child.props.children as ReactElement<AnyProps>[]).some(
        (route) => route?.props?.path !== undefined
      )
  ) as ReactElement<{ children: ReactElement<AnyProps>[] }>;

  const byPath: Record<string, AnyProps> = {};
  for (const route of Children.toArray(
    routesEl.props.children
  ) as ReactElement<AnyProps>[]) {
    const path = route.props.path as string | undefined;
    const element = route.props.element as ReactElement<AnyProps> | undefined;
    if (path && element) {
      byPath[path] = element.props;
    }
  }
  return byPath;
}

describe('BackupMongoApp route wiring', () => {
  it('wires the custom create and edit form slots on the restores tab', () => {
    const restores = schemaAppPropsByRoutePath()['restores/*'];

    expect(restores.renderCreateForm).toBe(restoreMongoCreateForm);
    expect(restores.renderEditForm).toBe(restoreMongoEditForm);
  });

  it('leaves the backups tab on the framework default edit renderer', () => {
    const backups = schemaAppPropsByRoutePath()['backups/*'];

    expect(backups.renderEditForm).toBeUndefined();
    expect(backups.renderCreateForm).toBeUndefined();
  });

  // Each plugin's routeBase must resolve under the path the host actually mounts
  // this app at, or the plugins' absolute nav (detail back/edit links, the
  // related-app tab bar) points outside the mount and 404s.
  it('derives every tab routeBase from the host-supplied basePath', () => {
    const byPath = schemaAppPropsByRoutePath();

    expect(byPath['config/*'].routeBase).toBe(`${BASE_PATH}/config`);
    expect(byPath['backups/*'].routeBase).toBe(`${BASE_PATH}/backups`);
    expect(byPath['restores/*'].routeBase).toBe(`${BASE_PATH}/restores`);
  });

  // The Configuration tab addresses the child app by its `key`
  // (`backup_mongo/config`), not its `name` (`backup_mongo_config`) -- the same
  // way the restores tab does. Getting this wrong yields an empty tab rather
  // than an error, because the schema fetch simply 404s.
  it('addresses the config child app by its key', () => {
    expect(schemaAppPropsByRoutePath()['config/*'].pluginName).toBe(
      'backup_mongo/config'
    );
  });

  // Nothing custom is passed for configuration: the app declares update=False /
  // delete=False so the framework derives the surface, and the form needs none of
  // the field overrides the backups and restores forms do. Asserted so that
  // adding one later is a deliberate act.
  it('leaves the config tab entirely on the framework defaults', () => {
    const props = schemaAppPropsByRoutePath()['config/*'];

    expect(props.renderField).toBeUndefined();
    expect(props.renderCreateForm).toBeUndefined();
    expect(props.renderEditForm).toBeUndefined();
    expect(props.renderTaskDetailChildren).toBeUndefined();
  });
});
