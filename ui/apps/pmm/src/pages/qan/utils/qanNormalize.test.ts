import {
  metricNamesFromResponse,
  parseQanColumns,
  asLabelValueList,
  asStringList,
} from './qanNormalize';
import { DEFAULT_QAN_COLUMNS } from './qanTools';

describe('qanNormalize', () => {
  describe('metricNamesFromResponse', () => {
    it('reads keys from API object map', () => {
      expect(
        metricNamesFromResponse({
          data: { load: 'Load', query_time: 'Query Time' },
        })
      ).toEqual(['load', 'query_time']);
    });

    it('supports legacy array shape', () => {
      expect(
        metricNamesFromResponse({
          data: [{ name: 'load', type: 'float' }],
        })
      ).toEqual(['load']);
    });
  });

  describe('parseQanColumns', () => {
    it('returns default for invalid JSON', () => {
      expect(parseQanColumns('not-json')).toEqual(DEFAULT_QAN_COLUMNS);
    });

    it('returns default for non-array JSON', () => {
      expect(parseQanColumns('{"load":1}')).toEqual(DEFAULT_QAN_COLUMNS);
    });

    it('parses string array from URL', () => {
      expect(parseQanColumns('["load","num_queries"]')).toEqual(['load', 'num_queries']);
    });
  });

  describe('asLabelValueList', () => {
    it('returns empty array for non-array filter names', () => {
      expect(asLabelValueList(undefined)).toEqual([]);
      expect(asLabelValueList('service')).toEqual([]);
    });
  });

  describe('asStringList', () => {
    it('coerces comma-separated string to array', () => {
      expect(asStringList('a,b')).toEqual(['a', 'b']);
    });
  });
});
