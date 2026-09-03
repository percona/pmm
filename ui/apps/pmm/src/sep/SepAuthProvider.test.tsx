import { render, screen } from '@testing-library/react';
import { useAuth } from '@sep/api';
import { User } from 'types/user.types';
import { SepAuthProvider } from './SepAuthProvider';

const useUserMock = vi.fn();

vi.mock('contexts/user', () => ({
  useUser: () => useUserMock(),
}));

/** Surfaces the capability the SEP framework and plugins read. */
const Probe = () => {
  const { isAdmin, canMutate } = useAuth();
  return (
    <div>
      <span data-testid="admin">{isAdmin ? 'yes' : 'no'}</span>
      <span data-testid="can-mutate">{canMutate ? 'yes' : 'no'}</span>
    </div>
  );
};

const renderProbe = () =>
  render(
    <SepAuthProvider>
      <Probe />
    </SepAuthProvider>
  );

beforeEach(() => {
  useUserMock.mockReset();
});

describe('SepAuthProvider', () => {
  it('grants mutation to a PMM admin', () => {
    useUserMock.mockReturnValue({ user: { isPMMAdmin: true } as User });

    renderProbe();

    expect(screen.getByTestId('admin')).toHaveTextContent('yes');
    expect(screen.getByTestId('can-mutate')).toHaveTextContent('yes');
  });

  it('withholds mutation from a non-admin', () => {
    useUserMock.mockReturnValue({ user: { isPMMAdmin: false } as User });

    renderProbe();

    expect(screen.getByTestId('admin')).toHaveTextContent('no');
    expect(screen.getByTestId('can-mutate')).toHaveTextContent('no');
  });

  it('withholds mutation while the PMM user is still loading', () => {
    useUserMock.mockReturnValue({ user: undefined });

    renderProbe();

    expect(screen.getByTestId('can-mutate')).toHaveTextContent('no');
  });
});

describe('useAuth outside a provider', () => {
  it('resolves to a non-admin session rather than throwing', () => {
    render(<Probe />);

    expect(screen.getByTestId('admin')).toHaveTextContent('no');
    expect(screen.getByTestId('can-mutate')).toHaveTextContent('no');
  });
});
