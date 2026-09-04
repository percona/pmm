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

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PropsWithChildren } from 'react';
import { TaskFilesDialog } from './TaskFilesDialog';

vi.mock('@sep/api', () => ({
  apiClient: {
    get: vi.fn(),
  },
  SEP_BASE_PATH: '/sep',
  // Stands in for the real recovery, which pulls axios into the graph this
  // manual factory exists to keep out. Every error here is a plain `Error` with
  // no blob body, which is exactly the case the real one passes through
  // untouched; the parsing path is covered in useTaskFileDownload.test.tsx.
  normalizeBlobError: async (error: unknown) => error,
}));

import { apiClient } from '@sep/api';

const mockedApiClient = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
};

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function Wrapper({
  children,
  client,
}: PropsWithChildren<{ client: QueryClient }>) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

const FILE_LIST = {
  'output/result.txt': { size: 1024, is_dir: false },
  'output/logs/': { size: 0, is_dir: true },
};

describe('TaskFilesDialog', () => {
  beforeEach(() => {
    mockedApiClient.get.mockReset();
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:mock');
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('does not render content when closed', () => {
    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open={false} taskHistoryId={1} onClose={vi.fn()} />
      </Wrapper>
    );
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('shows loading spinner while fetching', () => {
    mockedApiClient.get.mockImplementation(() => new Promise(() => {}));
    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={1} onClose={vi.fn()} />
      </Wrapper>
    );
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('renders file rows after fetch', async () => {
    mockedApiClient.get.mockResolvedValue({ data: FILE_LIST });
    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={1} onClose={vi.fn()} />
      </Wrapper>
    );
    await waitFor(() =>
      expect(screen.getByText('result.txt')).toBeInTheDocument()
    );
    expect(screen.getByText('1.0 KB')).toBeInTheDocument();
    expect(screen.getByText('logs')).toBeInTheDocument();
    expect(screen.getByText('Folder')).toBeInTheDocument();
    // Verify the list request escapes SEP's /api mount so it hits /sep/files/{id}
    expect(mockedApiClient.get).toHaveBeenCalledWith(
      '/files/1',
      expect.objectContaining({ baseURL: '/sep' })
    );
  });

  it('shows empty state when file list is empty', async () => {
    mockedApiClient.get.mockResolvedValue({ data: {} });
    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={1} onClose={vi.fn()} />
      </Wrapper>
    );
    await waitFor(() =>
      expect(screen.getByText(/No files available/i)).toBeInTheDocument()
    );
  });

  it('shows error alert when fetch fails', async () => {
    mockedApiClient.get.mockRejectedValue(new Error('Network Error'));
    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={1} onClose={vi.fn()} />
      </Wrapper>
    );
    await waitFor(() =>
      expect(screen.getByText(/Failed to load file list/i)).toBeInTheDocument()
    );
  });

  it('calls onClose when dialog is dismissed', async () => {
    mockedApiClient.get.mockResolvedValue({ data: FILE_LIST });
    const onClose = vi.fn();
    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={1} onClose={onClose} />
      </Wrapper>
    );
    await waitFor(() =>
      expect(screen.getByText('result.txt')).toBeInTheDocument()
    );
    await userEvent.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalled();
  });

  it('fires download mutation with correct path for a file', async () => {
    mockedApiClient.get
      .mockResolvedValueOnce({ data: FILE_LIST })
      .mockResolvedValueOnce({ data: new Blob(['content']) });

    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={42} onClose={vi.fn()} />
      </Wrapper>
    );

    await waitFor(() =>
      expect(screen.getByText('result.txt')).toBeInTheDocument()
    );

    const dialog = screen.getByRole('dialog');
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Download output/result.txt' })
    );

    await waitFor(() =>
      expect(mockedApiClient.get).toHaveBeenCalledWith(
        '/files/42/download',
        expect.objectContaining({
          baseURL: '/sep',
          params: { path: 'output/result.txt' },
          responseType: 'blob',
        })
      )
    );
  });

  it('fires download mutation with .tar.gz name for a directory', async () => {
    mockedApiClient.get
      .mockResolvedValueOnce({ data: FILE_LIST })
      .mockResolvedValueOnce({ data: new Blob(['archive']) });

    const capture = { anchor: null as HTMLAnchorElement | null };
    const origCreate = document.createElement.bind(document);
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = origCreate(tag);
      if (tag === 'a') {
        capture.anchor = el as HTMLAnchorElement;
      }
      return el;
    });

    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={42} onClose={vi.fn()} />
      </Wrapper>
    );

    await waitFor(() => expect(screen.getByText('logs')).toBeInTheDocument());

    const dialog = screen.getByRole('dialog');
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Download output/logs/' })
    );

    await waitFor(() =>
      expect(mockedApiClient.get).toHaveBeenCalledWith(
        '/files/42/download',
        expect.objectContaining({
          baseURL: '/sep',
          params: { path: 'output/logs/' },
          responseType: 'blob',
        })
      )
    );
    expect(capture.anchor?.download).toBe('logs.tar.gz');
  });

  it('uses full path in aria-label to disambiguate files with the same basename', async () => {
    const data = {
      'stdout/report.txt': { size: 100, is_dir: false },
      'stderr/report.txt': { size: 200, is_dir: false },
    };
    mockedApiClient.get.mockResolvedValue({ data });
    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={1} onClose={vi.fn()} />
      </Wrapper>
    );
    await waitFor(() =>
      expect(screen.getAllByText('report.txt')).toHaveLength(2)
    );
    expect(
      screen.getByRole('button', { name: 'Download stdout/report.txt' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Download stderr/report.txt' })
    ).toBeInTheDocument();
  });

  it('disables all download buttons while a download is in flight', async () => {
    let resolveDownload!: () => void;
    mockedApiClient.get
      .mockResolvedValueOnce({ data: FILE_LIST })
      .mockImplementationOnce(
        () =>
          new Promise<{ data: Blob }>((res) => {
            resolveDownload = () => res({ data: new Blob(['x']) });
          })
      );

    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={1} onClose={vi.fn()} />
      </Wrapper>
    );

    await waitFor(() =>
      expect(screen.getByText('result.txt')).toBeInTheDocument()
    );

    const dialog = screen.getByRole('dialog');
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Download output/result.txt' })
    );

    // All buttons disabled while download is pending
    await waitFor(() =>
      screen.getAllByRole('button', { name: /^Download/ }).forEach((btn) => {
        expect(btn).toBeDisabled();
      })
    );

    resolveDownload();
  });

  it('shows inline error alert on download failure', async () => {
    mockedApiClient.get
      .mockResolvedValueOnce({ data: FILE_LIST })
      .mockRejectedValueOnce(new Error('Server error'));

    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskFilesDialog open taskHistoryId={1} onClose={vi.fn()} />
      </Wrapper>
    );

    await waitFor(() =>
      expect(screen.getByText('result.txt')).toBeInTheDocument()
    );

    const dialog = screen.getByRole('dialog');
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Download output/result.txt' })
    );

    await waitFor(() =>
      expect(within(dialog).getByText('Server error')).toBeInTheDocument()
    );
  });
});
