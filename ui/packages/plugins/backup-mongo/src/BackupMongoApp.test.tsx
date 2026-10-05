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
import { showsList } from './ConfigTab';
import { CONFIG_APP_NAME } from './routes';
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
  // Search depth-first rather than scanning the root's direct children: the
  // routes sit inside ClusterScopeProvider, and any future wrapper would move
  // them again. What identifies <Routes> is that its children carry `path`.
  const findRoutes = (
    node: ReactElement<AnyProps>
  ): ReactElement<{ children: ReactElement<AnyProps>[] }> | undefined => {
    for (const child of Children.toArray(
      node?.props?.children as ReactElement<AnyProps>[] | undefined
    ) as ReactElement<AnyProps>[]) {
      if (!child?.props) {
        continue;
      }
      const kids = Children.toArray(
        child.props.children as ReactElement<AnyProps>[] | undefined
      ) as ReactElement<AnyProps>[];
      if (kids.some((route) => route?.props?.path !== undefined)) {
        return child as ReactElement<{ children: ReactElement<AnyProps>[] }>;
      }
      const nested = findRoutes(child);
      if (nested) {
        return nested;
      }
    }
    return undefined;
  };

  const routesEl = findRoutes(tree as ReactElement<AnyProps>) as ReactElement<{
    children: ReactElement<AnyProps>[];
  }>;

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

    // ConfigTab takes basePath and derives `${basePath}/config` itself, so the
    // guarantee moves with it -- asserted directly below.
    expect(byPath['config/*'].basePath).toBe(BASE_PATH);
    expect(byPath['backups/*'].routeBase).toBe(`${BASE_PATH}/backups`);
    expect(byPath['restores/*'].routeBase).toBe(`${BASE_PATH}/restores`);
  });

  // The Configuration tab addresses the child app by its `key`
  // (`backup_mongo/config`), not its `name` (`backup_mongo_config`) -- the same
  // way the restores tab does. Getting this wrong yields an empty tab rather
  // than an error, because the schema fetch simply 404s.
  it('addresses the config child app by its key', () => {
    // Asserted against the constant ConfigTab passes through, since the tab now
    // owns the SchemaDrivenPlugin rather than the route doing so.
    expect(CONFIG_APP_NAME).toBe('backup_mongo/config');
  });

  it('derives the config routeBase from basePath', () => {
    // The other half of the routeBase guarantee, now that ConfigTab builds it:
    // only the list collapses, so the derivation has to agree with the plugin's
    // own routes or the accordion would swallow the create form.
    expect(showsList(`${BASE_PATH}/config`, `${BASE_PATH}/config`)).toBe(true);
    expect(
      showsList(
        `${BASE_PATH}/config/backup_mongo_config`,
        `${BASE_PATH}/config`
      )
    ).toBe(true);
    expect(
      showsList(
        `${BASE_PATH}/config/backup_mongo_config/new`,
        `${BASE_PATH}/config`
      )
    ).toBe(false);
    expect(
      showsList(
        `${BASE_PATH}/config/backup_mongo_config/42`,
        `${BASE_PATH}/config`
      )
    ).toBe(false);
  });

  // Nothing custom is passed for configuration: the app declares update=False /
  // delete=False so the framework derives the surface, and the form needs none of
  // the field overrides the backups and restores forms do. Asserted so that
  // adding one later is a deliberate act.
  it('gives the config tab the cluster scope and nothing else', () => {
    // The config form carries the same Task-section trio as the other two
    // (task_name, service_id, hostname), so it needs the cluster to supply the
    // service and executor just as they do. Every other slot stays on the
    // framework defaults -- this app declares update=False/delete=False, so the
    // framework derives the rest of its surface.
    const props = schemaAppPropsByRoutePath()['config/*'];

    expect(props.renderField).toBeDefined();
    expect(props.renderCreateForm).toBeUndefined();
    expect(props.renderEditForm).toBeUndefined();
    expect(props.renderTaskDetailChildren).toBeUndefined();
  });
});
