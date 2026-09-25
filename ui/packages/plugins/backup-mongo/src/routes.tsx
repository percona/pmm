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

// SEP app names, as registered under SEP.APPS in SEP/settings.yaml. These address
// the backend and are not paths -- the mount path is the host's to choose and
// arrives as BackupMongoApp's `basePath` prop.
export const BACKUP_APP_NAME = 'backup_mongo';
export const RESTORE_APP_NAME = 'backup_mongo/restore';
// The child app's `key`, not its `name` (`backup_mongo_config`) -- the key is what
// addresses an app, as the restore entry above does.
export const CONFIG_APP_NAME = 'backup_mongo/config';
