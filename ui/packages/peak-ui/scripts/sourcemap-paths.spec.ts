// @vitest-environment node
import { describe, expect, it } from 'vitest';
import { stripPnpmStore } from './sourcemap-paths.mjs';

describe('stripPnpmStore', () => {
  it('drops the store folder, whose name carries peer versions', () => {
    expect(
      stripPnpmStore(
        '../../../node_modules/.pnpm/material-react-table@3.2.1_798f107b3e8f5f21af0f8517b7ad74fa/node_modules/material-react-table/dist/index.esm.js'
      )
    ).toBe('../../../node_modules/material-react-table/dist/index.esm.js');
  });
  it('keeps scoped package paths whole', () => {
    expect(
      stripPnpmStore(
        '../../../node_modules/.pnpm/@tanstack+react-virtual@3.11.2_react-dom@19.2.8_react@19.2.8__react@19.2.8/node_modules/@tanstack/react-virtual/dist/esm/index.js'
      )
    ).toBe('../../../node_modules/@tanstack/react-virtual/dist/esm/index.js');
  });
  it('leaves Peak UI sources alone', () => {
    expect(stripPnpmStore('../src/components/chip/chip.tsx')).toBe(
      '../src/components/chip/chip.tsx'
    );
  });
});
