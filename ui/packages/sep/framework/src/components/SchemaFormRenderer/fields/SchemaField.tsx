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

import { SchemaSelector } from '../../SchemaSelector';
import type { SchemaField as SchemaFieldType } from '../types';

interface SchemaFieldProps {
  field: SchemaFieldType;
}

export function SchemaField({ field }: SchemaFieldProps) {
  return (
    <SchemaSelector
      name={field.name}
      label={field.label}
      required={field.required}
      dependsOn={field.depends_on}
      allowCustom={field.allow_custom}
    />
  );
}
