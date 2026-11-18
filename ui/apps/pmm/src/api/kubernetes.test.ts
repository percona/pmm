import { hasKubernetesData } from './kubernetes';

const { get } = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock('./api', () => ({ grafanaApi: { get } }));

describe('hasKubernetesData', () => {
  beforeEach(() => {
    get.mockReset();
  });

  it('sends the presence query', async () => {
    get.mockResolvedValue({ data: { data: { result: [] } } });

    await hasKubernetesData();

    expect(get).toHaveBeenCalledWith(
      '/datasources/proxy/uid/PA58DA793C7250F1B/api/v1/query',
      { params: { query: 'group(kube_node_info{k8s_cluster_id!=""})' } }
    );
  });

  it('is true when a labelled cluster exists', async () => {
    get.mockResolvedValue({
      data: { data: { result: [{ metric: {}, value: [0, '1'] }] } },
    });

    expect(await hasKubernetesData()).toBe(true);
  });

  it('is false when there is no Kubernetes data', async () => {
    get.mockResolvedValue({ data: { data: { result: [] } } });

    expect(await hasKubernetesData()).toBe(false);
  });

  it('rejects when the request fails', async () => {
    get.mockRejectedValue(new Error('502'));

    await expect(hasKubernetesData()).rejects.toThrow('502');
  });
});
