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

import { HostSelector } from '../../HostSelector';
import { serviceTypesForDependsOn, useFormFields } from '../formFieldsContext';
import type { HostField as HostFieldType } from '../types';

interface HostFieldProps {
  field: HostFieldType;
}

export function HostField({ field }: HostFieldProps) {
  const fields = useFormFields();
  const serviceTypes = serviceTypesForDependsOn(fields, field.depends_on);

  return (
    <HostSelector
      name={field.name}
      label={field.label}
      required={field.required}
      dependsOn={field.depends_on}
      serviceTypes={serviceTypes}
      allowCustom={field.allow_custom}
    />
  );
}
