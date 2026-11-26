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

import { useEffect, useMemo, useRef } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import type { FormSection } from '../types';
import {
  collectOneOfGroups,
  flattenSectionFields,
  isOneOfGroup,
} from '../utils/flattenSectionFields';
import { watchValuesByName } from '../utils/watchValuesByName';

/**
 * Drop the leaves of every one-of group's inactive branches from the payload,
 * and return their names.
 *
 * Owned by the form body for the same reason as `useFieldPayloadCleanup`: a
 * group inside an unopened section never mounts its slot, whose own cleanup
 * would otherwise be the only one.
 */
export function useOneOfPayloadCleanup(sections: FormSection[]): string[] {
  const { control, unregister } = useFormContext();
  const groups = useMemo(() => collectOneOfGroups(sections), [sections]);
  const discriminators = useMemo(
    () => groups.map((group) => group.discriminator),
    [groups]
  );
  const rawModes = useWatch({
    control,
    name: discriminators,
    disabled: discriminators.length === 0,
  }) as unknown[];

  const inactive = useMemo(() => {
    const names: string[] = [];
    // Mirrors OneOfGroupSlot's choice of branch, fallback included.
    const modes = watchValuesByName(discriminators, rawModes);
    for (const group of groups) {
      const mode = String(
        modes[group.discriminator] ??
          group.default ??
          group.branches[0]?.value ??
          ''
      );
      const active =
        group.branches.find((branch) => branch.value === mode) ??
        group.branches[0];
      const activeNames = new Set(active?.fields.map((leaf) => leaf.name));
      for (const branch of group.branches) {
        for (const leaf of branch.fields) {
          if (!activeNames.has(leaf.name)) {
            names.push(leaf.name);
          }
        }
      }
    }
    return names;
  }, [groups, discriminators, rawModes]);

  // `useWatch` returns a fresh array every render; key the effect on content.
  const signature = inactive.join('|');
  useEffect(() => {
    for (const name of inactive) {
      unregister(name);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [signature, unregister]);

  return inactive;
}

/**
 * Names of the fields the current values leave out of the submitted payload,
 * and so out of the rendered form: every field of a gated-out section plus the
 * gate-hidden fields and inactive one-of leaves the cleanup hooks report.
 *
 * Read in render rather than from react-hook-form's registrations, which the
 * cleanup hooks only drop in an effect afterwards.
 */
export function useUnsubmittedFieldNames(
  sections: FormSection[],
  hiddenSections: boolean[],
  droppedFields: string[]
): Set<string> {
  const names = useMemo(() => {
    const unsubmitted = new Set(droppedFields);
    sections.forEach((section, i) => {
      if (!hiddenSections[i]) {
        return;
      }
      for (const field of flattenSectionFields([section])) {
        unsubmitted.add(field.name);
      }
      for (const field of section.fields) {
        if (isOneOfGroup(field)) {
          unsubmitted.add(field.discriminator);
        }
      }
    });
    return unsubmitted;
  }, [sections, hiddenSections, droppedFields]);

  // The cleanup hooks hand back fresh arrays every render; hold the previous
  // set while its members are unchanged so rule evaluation can memoise on it.
  const stable = useRef(names);
  if (
    stable.current.size !== names.size ||
    [...names].some((name) => !stable.current.has(name))
  ) {
    stable.current = names;
  }
  return stable.current;
}
