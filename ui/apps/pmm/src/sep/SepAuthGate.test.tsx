import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { ApiError, postSessionExchange, setTokenMinter } from '@sep/api';
import { SepAuthGate } from './SepAuthGate';
import { initSepAuth } from './bootstrap';
import { resetSepAuthStore } from './sepTokenStore';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  postSessionExchange: vi.fn(),
}));

const exchange = vi.mocked(postSessionExchange);

const unauthorized = () =>
  new ApiError({ kind: 'http', status: 401, message: 'no session' });

const renderGate = () =>
  render(
    <SepAuthGate>
      <div>plugin content</div>
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

describe('SepAuthGate', () => {
  it('withholds children until the exchange resolves', async () => {
    let resolveExchange: (value: {
      access_token: string;
      expires_in: number;
    }) => void = () => {};
    exchange.mockReturnValue(
      new Promise((resolve) => {
        resolveExchange = resolve;
      })
    );

    renderGate();

    expect(screen.queryByText('plugin content')).not.toBeInTheDocument();
    expect(screen.getByRole('progressbar')).toBeInTheDocument();

    resolveExchange({ access_token: 'bearer-1', expires_in: 300 });

    expect(await screen.findByText('plugin content')).toBeInTheDocument();
  });

  it('renders children once a bearer is held', async () => {
    exchange.mockResolvedValue({ access_token: 'bearer-1', expires_in: 300 });

    renderGate();

    expect(await screen.findByText('plugin content')).toBeInTheDocument();
    expect(exchange).toHaveBeenCalledOnce();
  });

  it('reports a rejected session instead of looping on the exchange', async () => {
    exchange.mockRejectedValue(unauthorized());

    renderGate();

    expect(await screen.findByTestId('sep-auth-error')).toHaveTextContent(
      'Not signed in'
    );
    expect(screen.queryByText('plugin content')).not.toBeInTheDocument();
    // Waiting past any plausible retry delay: the failure must stay put.
    await waitFor(() => expect(exchange).toHaveBeenCalledOnce());
    expect(exchange).toHaveBeenCalledOnce();
  });

  it('exchanges again when the user retries', async () => {
    exchange.mockRejectedValue(unauthorized());
    renderGate();
    await screen.findByTestId('sep-auth-error');

    exchange.mockResolvedValue({ access_token: 'bearer-1', expires_in: 300 });
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('plugin content')).toBeInTheDocument();
    expect(exchange).toHaveBeenCalledTimes(2);
  });

  it('distinguishes a transient failure from a rejected session', async () => {
    exchange.mockRejectedValue(new Error('network down'));

    renderGate();

    expect(await screen.findByTestId('sep-auth-error')).toHaveTextContent(
      'Could not reach Smart Expert Platform'
    );
  });
});
