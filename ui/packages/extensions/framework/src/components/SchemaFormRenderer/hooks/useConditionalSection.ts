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
import { isOneOfGroup } from '../utils/flattenSectionFields';
import {
  evaluatePredicate,
  getGateFieldNames,
} from '../utils/predicateEvaluator';
import { watchValuesByName } from '../utils/watchValuesByName';

export interface ConditionalSectionState {
  isHidden: boolean;
}

/**
 * Evaluate the `forbidden` gates of several sections against one watch.
 *
 * The batched form exists so a caller can know a whole run of sections is
 * hidden *before* rendering the shell that would contain them — a grouped
 * shell whose every member is gated out must not render as an empty
 * accordion. Subscribing once to the union of gate fields also keeps a form
 * with many gated sections to a single `useWatch`.
 *
 * Returns one flag per input section, positionally.
 *
 * Deliberately not exported from the package: it is only half the contract.
 * A caller that reads these flags without also calling
 * {@link useUnregisterHiddenSections} ships a hidden section's values in the
 * submission payload. {@link useConditionalSection} pairs the two for the
 * single-section case and is the supported entry point.
 */
export function useConditionalSections(sections: FormSection[]): boolean[] {
  const { control } = useFormContext();

  const watchedNames = useMemo<string[]>(() => {
    const names = new Set<string>();
    for (const section of sections) {
      for (const name of getGateFieldNames(section.forbidden ?? [])) {
        names.add(name);
      }
    }
    return [...names];
    // section schemas are static for the form's lifetime; dep on sections is safe.
  }, [sections]);

  const rawValues = useWatch({
    control,
    name: watchedNames,
    disabled: watchedNames.length === 0,
  }) as unknown[];

  const flags = useMemo(() => {
    const map = watchValuesByName(watchedNames, rawValues);
    return sections.map((section) =>
      Boolean(
        section.forbidden?.some((gate) => evaluatePredicate(gate.when, map))
      )
    );
  }, [sections, watchedNames, rawValues]);

  // `useWatch` hands back a fresh array every render, so the memo above
  // recomputes every render too. Hold the previous result while the flags
  // themselves are unchanged, so callers can put the array straight into an
  // effect's dependency list without that effect re-firing continuously.
  const stable = useRef<boolean[]>(flags);
  if (
    stable.current.length !== flags.length ||
    flags.some((flag, i) => stable.current[i] !== flag)
  ) {
    stable.current = flags;
  }
  return stable.current;
}

/** The react-hook-form names every field in a section registers under. */
function sectionFieldNames(section: FormSection): string[] {
  const names = new Set<string>();
  for (const field of section.fields) {
    if (isOneOfGroup(field)) {
      names.add(field.name);
      names.add(field.discriminator);
      for (const branch of field.branches) {
        for (const leaf of branch.fields) {
          names.add(leaf.name);
        }
      }
      continue;
    }
    names.add(field.name);
  }
  return [...names];
}

/**
 * Drop every hidden section's fields from react-hook-form state.
 *
 * Owned by the form body rather than by each section's own component, because
 * a section that never mounts cannot clean up after itself — a member of a
 * collapsed group is not in the tree at all, and would otherwise ship its
 * default value while gated out.
 */
export function useUnregisterHiddenSections(
  sections: FormSection[],
  hidden: boolean[]
): void {
  const { unregister } = useFormContext();

  const namesBySection = useMemo(
    () => sections.map(sectionFieldNames),
    [sections]
  );

  useEffect(() => {
    hidden.forEach((isHidden, i) => {
      if (!isHidden) {
        return;
      }
      for (const name of namesBySection[i] ?? []) {
        unregister(name);
      }
    });
  }, [hidden, namesBySection, unregister]);
}

/**
 * Evaluate one section's gates and drop its fields while it is hidden.
 *
 * When the section becomes hidden every child field must drop out of
 * RHF state so its (possibly default) value does not ship. On re-show,
 * the renderer re-mounts the children which re-call register() and
 * start fresh — intentional: stale user input from the prior session
 * would otherwise ship in the payload and fail backend cross-mode validation.
 */
export function useConditionalSection(
  section: FormSection
): ConditionalSectionState {
  const sections = useMemo(() => [section], [section]);
  const hidden = useConditionalSections(sections);

  useUnregisterHiddenSections(sections, hidden);

  return { isHidden: hidden[0] ?? false };
}
