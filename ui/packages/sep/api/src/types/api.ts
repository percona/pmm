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

// Common API types re-exported from the OpenAPI-generated schemas.
// Hand-written types were removed after SEP-963 Phase 2 codegen landed.
// The runtime ApiError class lives in ../errors.ts.

import type { components } from '../generated/main';

/** Authenticated user profile — `GET /api/users/me`. */
export type User = components['schemas']['CasdoorUser'];

/**
 * Slim OAuth token returned to SPA clients by `POST /api/oauth/login` and
 * `POST /api/oauth/refresh`. Only contains the access token and its TTL —
 * the refresh token rides an `HttpOnly` cookie and is opaque to JS.
 */
export type SPAOAuthTokenResponse =
  components['schemas']['SPAOAuthTokenResponse'];

/**
 * @deprecated Use `SPAOAuthTokenResponse`. The full `OAuthToken` shape is
 * only returned by the legacy `/oauth/token` endpoint; the SPA uses the
 * cookie-based `/oauth/login` + `/oauth/refresh` flow.
 */
export type OAuthTokenResponse = SPAOAuthTokenResponse;
