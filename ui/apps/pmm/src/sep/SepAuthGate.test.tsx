import { act, fireEvent, render, screen } from '@testing-library/react';
import { ApiError, postSessionExchange, setTokenMinter } from '@sep/api';
import { SepAuthGate } from './SepAuthGate';
import { initSepAuth } from './bootstrap';
import { markSepSignedOut, resetSepAuthStore } from './sepTokenStore';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  postSessionExchange: vi.fn(),
}));

const exchange = vi.mocked(postSessionExchange);

const bearer = (accessToken = 'bearer-1') => ({
  access_token: accessToken,
  expires_in: 300,
});

const unauthorized = () =>
  new ApiError({ kind: 'http', status: 401, message: 'no session' });

const renderGate = () =>
  render(
    <SepAuthGate>
      <div>plugin content</div>
    </SepAuthGate>
  );

/** A page with unsaved input, standing in for a half-filled plugin form. */
const renderGateWithForm = () =>
  render(
    <SepAuthGate>
      <input aria-label="target" defaultValue="" />
    </SepAuthGate>
  );

beforeEach(() => {
  exchange.mockReset();
  resetSepAuthStore();
  initSepAuth();
});

afterEach(() => {
  resetSepAuthStore();
  setTokenMinter(null);
});

describe('SepAuthGate — bootstrap', () => {
  it('withholds children until the exchange resolves', async () => {
    let resolveExchange: (value: ReturnType<typeof bearer>) => void = () => {};
    exchange.mockReturnValue(
      new Promise((resolve) => {
        resolveExchange = resolve;
      })
    );

    renderGate();

    expect(screen.queryByText('plugin content')).not.toBeInTheDocument();
    expect(screen.getByRole('progressbar')).toBeInTheDocument();

    resolveExchange(bearer());

    expect(await screen.findByText('plugin content')).toBeInTheDocument();
  });

  it('renders children once a bearer is held', async () => {
    exchange.mockResolvedValue(bearer());

    renderGate();

    expect(await screen.findByText('plugin content')).toBeInTheDocument();
    expect(exchange).toHaveBeenCalledOnce();
  });

  it('shows a signed-out page instead of the plugin, and does not loop', async () => {
    exchange.mockRejectedValue(unauthorized());

    renderGate();

    expect(await screen.findByTestId('sep-auth-error')).toHaveTextContent(
      'Not signed in'
    );
    expect(screen.queryByText('plugin content')).not.toBeInTheDocument();
    expect(exchange).toHaveBeenCalledOnce();
  });

  it('distinguishes an unreachable SEP from a rejected session', async () => {
    exchange.mockRejectedValue(new Error('network down'));

    renderGate();

    expect(await screen.findByTestId('sep-auth-error')).toHaveTextContent(
      "This page can't be loaded"
    );
  });

  it('exchanges again when the user retries', async () => {
    exchange.mockRejectedValue(unauthorized());
    renderGate();
    await screen.findByTestId('sep-auth-error');

    exchange.mockResolvedValue(bearer());
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('plugin content')).toBeInTheDocument();
    expect(exchange).toHaveBeenCalledTimes(2);
  });
});

describe('SepAuthGate — failure on a mounted page', () => {
  it('reports a rejected session without unmounting the page', async () => {
    exchange.mockResolvedValue(bearer());
    renderGate();
    await screen.findByText('plugin content');

    act(() => markSepSignedOut());

    expect(screen.getByTestId('sep-auth-notice')).toBeInTheDocument();
    expect(screen.getByText('plugin content')).toBeInTheDocument();
    expect(screen.queryByTestId('sep-auth-error')).not.toBeInTheDocument();
  });

  it('preserves in-progress form state', async () => {
    exchange.mockResolvedValue(bearer());
    renderGateWithForm();
    const field = await screen.findByLabelText('target');
    fireEvent.change(field, { target: { value: 'half-written command' } });

    act(() => markSepSignedOut());

    expect(screen.getByTestId('sep-auth-notice')).toBeInTheDocument();
    expect(screen.getByLabelText('target')).toHaveValue('half-written command');
  });

  it('clears the notice when the retry succeeds, keeping the page throughout', async () => {
    exchange.mockResolvedValue(bearer());
    renderGateWithForm();
    const field = await screen.findByLabelText('target');
    fireEvent.change(field, { target: { value: 'half-written command' } });
    act(() => markSepSignedOut());

    exchange.mockResolvedValue(bearer('bearer-2'));
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));

    await screen.findByLabelText('target');
    expect(screen.queryByTestId('sep-auth-notice')).not.toBeInTheDocument();
    expect(screen.getByLabelText('target')).toHaveValue('half-written command');
  });
});
