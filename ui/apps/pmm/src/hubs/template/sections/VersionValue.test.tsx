import { fireEvent, screen } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import { minutesAgo } from '../hub.testFixtures';
import { VersionValue } from './VersionValue';

describe('VersionValue', () => {
  it('shows a version alone, with nothing to focus, while there is nothing more to say', () => {
    renderHub(<VersionValue version={{ effective: '8.0.32-14' }} />);

    expect(screen.getByText('8.0.32-14')).not.toHaveAttribute('tabindex');
    expect(screen.queryByRole('img')).toBeNull();
  });

  it('marks mixed versions with a warning, in words for assistive tech, and lists them in a focusable tooltip', async () => {
    renderHub(
      <VersionValue
        version={{
          effective: '7.0.24-13',
          drift: [
            { version: '7.0.24-13', members: ['catalog-m1'] },
            { version: '7.0.30-16', members: ['catalog-m2', 'catalog-m3'] },
          ],
        }}
      />
    );

    const value = screen.getByText(/7\.0\.24-13/).closest('[tabindex="0"]');

    expect(value).toHaveTextContent('7.0.24-13 mixed versions');
    fireEvent.mouseOver(value as Element);
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Mixed versions: 7.0.24-13 on catalog-m1; 7.0.30-16 on catalog-m2, catalog-m3'
    );
  });

  it('says an update is available beneath the version, and to which release', async () => {
    renderHub(
      <VersionValue
        version={{
          effective: '7.0.24-13',
          update: {
            state: 'available',
            latest: '7.0.30-16',
            checkedAt: minutesAgo(120),
          },
        }}
      />
    );

    fireEvent.mouseOver(screen.getByText('Update available'));
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      '7.0.30-16 is available, checked 2h ago'
    );
  });

  it('reads a failed or stale check as Check failed, never as up to date', async () => {
    const { unmount } = renderHub(
      <VersionValue
        version={{
          effective: '7.0.30-16',
          update: {
            state: 'unknown',
            reason: 'stale',
            checkedAt: minutesAgo(3 * 24 * 60),
          },
        }}
      />
    );

    fireEvent.mouseOver(screen.getByText('Check failed'));
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'The version check last ran 3d ago, too long ago to trust'
    );
    unmount();

    renderHub(
      <VersionValue
        version={{
          effective: '7.0.30-16',
          update: { state: 'upToDate', checkedAt: minutesAgo(120) },
        }}
      />
    );
    expect(screen.queryByText('Check failed')).toBeNull();
    expect(screen.queryByText('Update available')).toBeNull();
    fireEvent.mouseOver(screen.getByText('7.0.30-16'));
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Up to date, checked 2h ago'
    );
  });
});
