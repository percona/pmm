import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook } from '@testing-library/react';
import { NavigationContext } from 'contexts/navigation';
import { TourContext, type TourContextProps } from 'contexts/tour';
import { UserContext } from 'contexts/user';
import type { PropsWithChildren } from 'react';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { navTreeFor } from './search.testFixtures';
import { useSearchIndex } from './useSearchIndex';

describe('useSearchIndex', () => {
  it('keeps the index when startTour changes, and runs the latest one', () => {
    const queryClient = new QueryClient();
    const navTree = navTreeFor(TEST_USER_ADMIN);
    const firstStartTour = vi.fn();
    let tour: TourContextProps = {
      startTour: firstStartTour,
      endTour: vi.fn(),
    };
    const { result, rerender } = renderHook(() => useSearchIndex(false), {
      wrapper: ({ children }: PropsWithChildren) => (
        <QueryClientProvider client={queryClient}>
          <UserContext.Provider
            value={{ isLoading: false, user: TEST_USER_ADMIN }}
          >
            <NavigationContext.Provider
              value={{ navTree, navOpen: true, setNavOpen: vi.fn() }}
            >
              <TourContext.Provider value={tour}>
                {children}
              </TourContext.Provider>
            </NavigationContext.Provider>
          </UserContext.Provider>
        </QueryClientProvider>
      ),
    });
    const { staticItems, items } = result.current;
    const latestStartTour = vi.fn();

    tour = { ...tour, startTour: latestStartTour };
    rerender();
    result.current.staticItems
      .find((item) => item.id === 'action:start-tour')
      ?.perform?.();

    expect(result.current.staticItems).toBe(staticItems);
    expect(result.current.items).toBe(items);
    expect(latestStartTour).toHaveBeenCalledWith('product');
    expect(firstStartTour).not.toHaveBeenCalled();
  });
});
