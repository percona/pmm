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

export type DipperCollectorType = 'environment' | 'pmm';

export interface DipperExecutionWrite {
  service_id: number;
  collector_type: DipperCollectorType;
  executor_host: string;
  sudo: boolean;
  args: Record<string, unknown>;
}

export interface DipperExecutionResponse {
  task_id: number | null;
  task_name: string;
  snippet_filename: string;
  service_id: number;
  collector_type: DipperCollectorType;
}
