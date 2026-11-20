import { act, render, screen } from '@testing-library/react';
import { useScrollEdges } from './useScrollEdges';

let report: IntersectionObserverCallback;

class IntersectionObserverMock {
  constructor(callback: IntersectionObserverCallback) {
    report = callback;
  }
  observe = vi.fn();
  disconnect = vi.fn();
}

const Scroller = ({ listKey = 'list' }: { listKey?: string }) => {
  const { rootRef, startRef, endRef, canScrollUp, canScrollDown } =
    useScrollEdges<HTMLUListElement, HTMLLIElement>();

  return (
    <>
      <ul key={listKey} ref={rootRef}>
        <li ref={startRef} data-testid="start" />
        <li ref={endRef} data-testid="end" />
      </ul>
      <span>{`up:${canScrollUp} down:${canScrollDown}`}</span>
    </>
  );
};

const showMarker = (testId: string, isIntersecting: boolean) =>
  act(() =>
    report(
      [{ target: screen.getByTestId(testId), isIntersecting }] as never,
      {} as IntersectionObserver
    )
  );

describe('useScrollEdges', () => {
  beforeEach(() => {
    vi.stubGlobal('IntersectionObserver', IntersectionObserverMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('reports each direction that has content out of view', () => {
    render(<Scroller />);

    showMarker('start', true);
    showMarker('end', false);
    expect(screen.getByText('up:false down:true')).toBeInTheDocument();

    showMarker('start', false);
    showMarker('end', true);
    expect(screen.getByText('up:true down:false')).toBeInTheDocument();
  });

  it('follows a list that remounts', () => {
    const { rerender } = render(<Scroller listKey="first" />);
    const removedStart = screen.getByTestId('start');

    rerender(<Scroller listKey="second" />);
    act(() =>
      report(
        [{ target: removedStart, isIntersecting: false }] as never,
        {} as IntersectionObserver
      )
    );
    showMarker('start', true);
    showMarker('end', true);

    expect(screen.getByText('up:false down:false')).toBeInTheDocument();
  });
});
