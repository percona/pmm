import { fireEvent, render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { RealtimeSessionStatus } from 'types/rta.types';
import type { SessionRow } from '../SessionsTable.types';
import SessionStatus from './SessionStatus';

const row = (overrides: Partial<SessionRow>): SessionRow => ({
  sessionId: 'service-1',
  sessionName: 'Service 1',
  type: 'service',
  startTime: new Date().toISOString(),
  status: RealtimeSessionStatus.error,
  serviceSessions: [],
  ...overrides,
});

const renderStatus = (session: SessionRow) =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <SessionStatus session={session} />
    </ThemeProvider>
  );

describe('SessionStatus', () => {
  it('explains an error with the reason the agent reported', async () => {
    renderStatus(
      row({
        statusMessage:
          'Real-Time Analytics is not supported for this instance: performance_schema is disabled',
      })
    );

    expect(screen.getByText('Error')).toBeInTheDocument();
    fireEvent.mouseOver(screen.getByTestId('session-status-message'));

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'performance_schema is disabled'
    );
  });

  it('shows a bare error when the agent sent no reason', () => {
    renderStatus(row({}));

    expect(screen.getByText('Error')).toBeInTheDocument();
    expect(
      screen.queryByTestId('session-status-message')
    ).not.toBeInTheDocument();
  });

  it('flags a running session that cannot collect everything', async () => {
    renderStatus(
      row({
        status: RealtimeSessionStatus.running,
        statusMessage:
          'The wait/lock/metadata/sql/mdl instrument is disabled, so metadata lock waits cannot be detected.',
      })
    );

    expect(screen.getByText(/Running for/)).toBeInTheDocument();
    fireEvent.mouseOver(screen.getByTestId('session-status-message'));

    const tooltip = await screen.findByRole('tooltip');
    expect(tooltip).toHaveTextContent('some details cannot be collected');
    expect(tooltip).toHaveTextContent('wait/lock/metadata/sql/mdl');
  });

  it('keeps a healthy running session plain', () => {
    renderStatus(row({ status: RealtimeSessionStatus.running }));

    expect(screen.getByText(/Running for/)).toBeInTheDocument();
    expect(
      screen.queryByTestId('session-status-message')
    ).not.toBeInTheDocument();
  });
});

describe('SessionStatus warnings', () => {
  it('lists each finding apart in a scrollable tooltip', async () => {
    renderStatus(
      row({
        status: RealtimeSessionStatus.running,
        statusMessage:
          'The events_statements_current consumer is disabled, so rows examined are not collected.\nThe wait/lock/metadata/sql/mdl instrument is disabled, so metadata lock waits cannot be detected.',
      })
    );

    fireEvent.mouseOver(screen.getByTestId('session-status-message'));

    const content = await screen.findByTestId('session-status-message-tooltip');
    const items = content.querySelectorAll('li');
    expect(items).toHaveLength(2);
    expect(items[0]).toHaveTextContent('events_statements_current');
    expect(items[1]).toHaveTextContent('wait/lock/metadata/sql/mdl');
    expect(content).toHaveStyle({ overflowY: 'auto' });
  });
});
