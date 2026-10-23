import * as Peak from '@percona/peak-ui';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render } from '@testing-library/react';
import type { PropsWithChildren, ReactElement } from 'react';
import { useLocation } from 'react-router-dom';
import type { Settings } from 'types/settings.types';
import type { User } from 'types/user.types';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { wrapWithSettings } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';

// The published Peak UI 1.0.26 has none of these; a local Peak UI with the hub's branches has them all.
export const hasPeakHub = [
  'PageHeader',
  'EmptyState',
  'StatusBanner',
  'StatusCard',
  'StateIndicator',
].every((name) => (Peak as unknown as Record<string, unknown>)[name] != null);

// Shows the router location, `/mongodb?hubState=unhealthy#clusters`, for assertions on URL writes.
const LocationProbe = () => {
  const { pathname, search, hash } = useLocation();

  return <output data-testid="hub-location">{pathname + search + hash}</output>;
};

/** The nearest ancestor matching the selector; fails the test when there is none. */
export const closestTo = (element: Element | null, selector: string) => {
  const match = element?.closest<HTMLElement>(selector);

  if (!match) {
    throw new Error(`No ${selector} around the element`);
  }

  return match;
};

export interface RenderHubOptions {
  /** Router path with its query, such as `/mongodb?hubState=unhealthy`. */
  path?: string;
  user?: User;
  settings?: Partial<Settings>;
}

/** Renders hub UI with the router, user, settings, query client and Peak theme. */
export const renderHub = (
  ui: ReactElement,
  {
    path = '/mongodb',
    user = TEST_USER_ADMIN,
    settings = {},
  }: RenderHubOptions = {}
) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(ui, {
    wrapper: ({ children }: PropsWithChildren) => (
      <TestWrapper
        userContext={{ isLoading: false, user }}
        routerProps={{ initialEntries: [path] }}
      >
        {wrapWithSettings(
          <QueryClientProvider client={queryClient}>
            {children}
            <LocationProbe />
          </QueryClientProvider>,
          { settings }
        )}
      </TestWrapper>
    ),
  });
};
