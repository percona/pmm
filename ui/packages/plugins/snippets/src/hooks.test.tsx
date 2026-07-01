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

import type { ReactNode } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { apiClient, setTokenProvider } from '@sep/api';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useSnippetDownload } from './hooks';

type CapturedHeaders = {
  Authorization?: string;
  get?: (name: string) => string | null | undefined;
  [key: string]: unknown;
};
interface CapturedRequestConfig {
  url?: string;
  method?: string;
  responseType?: string;
  headers?: CapturedHeaders;
}

function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
  };
}

const originalAdapter = apiClient.defaults.adapter;
const originalCreateObjectURL = URL.createObjectURL;
const originalRevokeObjectURL = URL.revokeObjectURL;

describe('useSnippetDownload', () => {
  let lastConfig: CapturedRequestConfig | null = null;
  let blobBody: Blob;
  let createObjectSpy: ReturnType<typeof vi.fn>;
  let revokeObjectSpy: ReturnType<typeof vi.fn>;
  let clickSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    lastConfig = null;
    blobBody = new Blob(['#!/bin/sh\necho hi\n'], {
      type: 'text/x-shellscript',
    });
    (apiClient.defaults as unknown as { adapter: unknown }).adapter = (
      config: CapturedRequestConfig
    ) => {
      lastConfig = config;
      return Promise.resolve({
        data: blobBody,
        status: 200,
        statusText: 'OK',
        headers: {
          'content-type': 'text/x-shellscript',
          'content-disposition': 'attachment; filename="hello.sh"',
        },
        config,
        request: {},
      });
    };

    setTokenProvider(() => 'test-access-token');

    createObjectSpy = vi.fn(() => 'blob:mock-url');
    revokeObjectSpy = vi.fn();
    Object.defineProperty(URL, 'createObjectURL', {
      value: createObjectSpy,
      writable: true,
      configurable: true,
    });
    Object.defineProperty(URL, 'revokeObjectURL', {
      value: revokeObjectSpy,
      writable: true,
      configurable: true,
    });

    clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => {});
  });

  afterEach(() => {
    (apiClient.defaults as unknown as { adapter: unknown }).adapter =
      originalAdapter;
    setTokenProvider(() => null);
    Object.defineProperty(URL, 'createObjectURL', {
      value: originalCreateObjectURL,
      writable: true,
      configurable: true,
    });
    Object.defineProperty(URL, 'revokeObjectURL', {
      value: originalRevokeObjectURL,
      writable: true,
      configurable: true,
    });
    clickSpy.mockRestore();
  });

  it('GETs /apps/snippets/snippet/download?snippet_filename=... with Bearer auth and a blob responseType', async () => {
    const { result } = renderHook(() => useSnippetDownload('hello.sh'), {
      wrapper: makeWrapper(),
    });

    await act(async () => {
      result.current.mutate();
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(lastConfig).not.toBeNull();
    const captured = lastConfig as CapturedRequestConfig;
    expect(captured.url).toBe(
      '/apps/snippets/snippet/download?snippet_filename=hello.sh'
    );
    expect(captured.method?.toLowerCase()).toBe('get');
    expect(captured.responseType).toBe('blob');

    const headers = captured.headers;
    const auth =
      typeof headers?.get === 'function'
        ? headers.get('Authorization')
        : headers?.Authorization;
    expect(auth).toBe('Bearer test-access-token');
  });

  it('encodes nested filenames in the query string', async () => {
    const nested = 'diag/slow-query.sh';
    const { result } = renderHook(() => useSnippetDownload(nested), {
      wrapper: makeWrapper(),
    });

    await act(async () => {
      result.current.mutate();
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    const captured = lastConfig as CapturedRequestConfig;
    expect(captured.url).toBe(
      '/apps/snippets/snippet/download?snippet_filename=diag%2Fslow-query.sh'
    );
    const [path] = (captured.url ?? '').split('?');
    expect(path).not.toContain('%2F');
    expect(path).not.toContain('diag');
  });

  it('reads the response body as a Blob and triggers a download with the snippet filename', async () => {
    const { result } = renderHook(() => useSnippetDownload('long-script.sh'), {
      wrapper: makeWrapper(),
    });

    let downloadAttr: string | null = null;
    let hrefAttr: string | null = null;
    clickSpy.mockImplementation(function clickImpl(this: HTMLAnchorElement) {
      downloadAttr = this.download;
      hrefAttr = this.href;
    });

    await act(async () => {
      result.current.mutate();
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toBeInstanceOf(Blob);
    expect(createObjectSpy).toHaveBeenCalledTimes(1);
    expect(createObjectSpy).toHaveBeenCalledWith(expect.any(Blob));
    expect(clickSpy).toHaveBeenCalledTimes(1);
    expect(downloadAttr).toBe('long-script.sh');
    expect(hrefAttr).toContain('blob:mock-url');

    await waitFor(() => {
      expect(revokeObjectSpy).toHaveBeenCalledWith('blob:mock-url');
    });
  });
});
