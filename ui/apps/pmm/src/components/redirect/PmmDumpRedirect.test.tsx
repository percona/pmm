import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import PmmDumpRedirect from './PmmDumpRedirect';

const LocationProbe = () => {
  const { pathname, search, hash } = useLocation();

  return <span data-testid="location-probe">{pathname + search + hash}</span>;
};

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="graph/pmm-dump/*" element={<PmmDumpRedirect />} />
        <Route path="pmm-dump" element={<div>Dump list</div>} />
        <Route path="pmm-dump/new" element={<div>New dataset</div>} />
        <Route path="pmm-dump/*" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>
  );

describe('PmmDumpRedirect', () => {
  it('redirects the old Grafana list route', () => {
    renderAt('/graph/pmm-dump');

    expect(screen.getByText('Dump list')).toBeInTheDocument();
  });

  it('keeps the subpath', () => {
    renderAt('/graph/pmm-dump/new');

    expect(screen.getByText('New dataset')).toBeInTheDocument();
  });

  it('keeps the query string and hash', () => {
    renderAt('/graph/pmm-dump/logs?orgId=1#top');

    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      '/pmm-dump/logs?orgId=1#top'
    );
  });
});
