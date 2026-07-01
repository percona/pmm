import { PMM_BASE_PATH } from 'lib/constants';
import { stripQanServiceId } from 'utils/qanServiceId';

export function buildNativeQanPath(params: Record<string, string | undefined>): string {
  const sp = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v == null || v === '') return;
    if (k === 'serviceId' || k === 'service_id') {
      const id = stripQanServiceId(v);
      sp.set('filter_service_id', id);
      sp.set('service_id', id);
      return;
    }
    sp.set(k, v);
  });
  const q = sp.toString();
  return `${PMM_BASE_PATH}/qan${q ? `?${q}` : ''}`;
}
