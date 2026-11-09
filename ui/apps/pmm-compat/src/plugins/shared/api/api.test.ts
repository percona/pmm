import { getBackendSrv } from '@grafana/runtime';
import { get, post } from './api';

jest.mock('@grafana/runtime', () => ({ getBackendSrv: jest.fn() }));

const getMock = jest.fn();
const postMock = jest.fn();

describe('api', () => {
  beforeEach(() => {
    getMock.mockReset().mockResolvedValue({ ok: true });
    postMock.mockReset().mockResolvedValue({ ok: true });
    jest.mocked(getBackendSrv).mockReturnValue({
      get: getMock,
      post: postMock,
    } as unknown as ReturnType<typeof getBackendSrv>);
  });

  it('should GET from the PMM API without error alerts', async () => {
    expect(await get('server/updates', { force: false })).toEqual({
      ok: true,
    });
    expect(getMock).toHaveBeenCalledWith(
      '/v1/server/updates',
      { force: false },
      undefined,
      { showErrorAlert: false }
    );
  });

  it('should POST to the PMM API without error alerts', async () => {
    expect(await post('actions:startNodeAction', { node_id: 'n1' })).toEqual({
      ok: true,
    });
    expect(postMock).toHaveBeenCalledWith(
      '/v1/actions:startNodeAction',
      { node_id: 'n1' },
      { showErrorAlert: false }
    );
  });
});
