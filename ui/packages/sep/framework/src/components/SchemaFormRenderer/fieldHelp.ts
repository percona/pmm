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

import type { PluginField } from './types';

/**
 * Longest description still shown under the input by default.
 *
 * Roughly one rendered line at the form's 800px width. The point of the
 * boundary is that a hint short enough to sit on one line costs almost nothing
 * to leave visible, while prose that wraps two or three times is what turns a
 * form with sixty fields into several screens.
 */
export const INLINE_HELP_MAX_LENGTH = 120;

/**
 * Whether a field's description belongs under the input rather than behind the
 * help icon beside its label.
 *
 * The schema decides when it says so; otherwise length does. Keeping the
 * default automatic matters because most apps never set the key: a short
 * format hint stays visible for them without anyone having to annotate it.
 */
export function isInlineHelp(field: PluginField): boolean {
  if (field.help_placement) {
    return field.help_placement === 'inline';
  }
  return (field.description?.length ?? 0) <= INLINE_HELP_MAX_LENGTH;
}

/**
 * Split a field's description into the two slots a renderer can put it in.
 *
 * Every field kind that can show help both ways goes through this, so the
 * policy lives in one place instead of being decided again in each of a dozen
 * renderers — which is how the description came to be rendered twice on some
 * kinds and not at all on others.
 *
 * @returns `tooltip` for the help icon beside the label, `inline` for text
 * under the input. At most one is ever set.
 */
export function fieldHelp(field: PluginField): {
  tooltip?: string;
  inline?: string;
} {
  const description = field.description;
  if (!description) {
    return {};
  }
  return isInlineHelp(field)
    ? { inline: description }
    : { tooltip: description };
}
