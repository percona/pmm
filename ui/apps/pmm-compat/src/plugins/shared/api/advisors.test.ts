import { get } from './api';
import { getFailedServices } from './advisors';

jest.mock('./api');

describe('getFailedServices', () => {
  it('should map counts to numbers and sort by service name', async () => {
    jest.mocked(get).mockResolvedValue({
      result: [
        { service_name: 'b', service_id: '2', critical_count: '3' },
        { service_name: 'a', service_id: '1', warning_count: '1' },
      ],
    });

    const result = await getFailedServices();

    expect(get).toHaveBeenCalledWith('advisors/failedServices');
    expect(result.map(({ serviceName }) => serviceName)).toEqual(['a', 'b']);
    expect(result[1].counts).toEqual({
      emergency: 0,
      alert: 0,
      critical: 3,
      error: 0,
      warning: 0,
      notice: 0,
      info: 0,
      debug: 0,
    });
  });

  it('should return an empty list without results', async () => {
    jest.mocked(get).mockResolvedValue({});

    expect(await getFailedServices()).toEqual([]);
  });
});
