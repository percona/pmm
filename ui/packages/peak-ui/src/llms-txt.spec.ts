import { existsSync, readFileSync } from 'node:fs';
import { posix, resolve } from 'node:path';
import { sanitize, storyNameFromExport, toId } from 'storybook/internal/csf';
import ts from 'typescript';
import { describe, expect, it } from 'vitest';
import { MATURITY_TAGS } from '../.storybook/maturity-tags';
import { DEFAULT_PAGE_MAX_WIDTH, DEFAULT_TABLE_STATE } from './index';

const root = process.cwd();
const read = (file: string) => readFileSync(resolve(root, file), 'utf8');
const llms = read('llms.txt');
const lines = llms.split('\n');
const pkg = JSON.parse(read('package.json')) as { files: string[] };

const STORYBOOK = 'https://percona.github.io/peak-ui/?path=/';
const SOURCE = 'https://github.com/percona/peak-ui/blob/main/';
const LINK_ITEM = /^- \[[^\]]+\]\(https?:\/\/[^)\s]+\): \S/;

const links = [...llms.matchAll(/\[[^\]]+\]\((https?:\/\/[^)\s]+)\)/g)].map((m) => m[1]);
const lineFor = (name: string) => lines.find((line) => line.startsWith(`- [${name}](`)) ?? '';
const section = (heading: string) => {
  const start = lines.indexOf(`## ${heading}`);
  const end = lines.findIndex((line, i) => i > start && line.startsWith('## '));
  return lines.slice(start + 1, end === -1 ? undefined : end);
};

// Every name the entry point exports, values and types alike, as the compiler sees it.
const publicExports = (() => {
  const configPath = ts.findConfigFile(root, ts.sys.fileExists, 'tsconfig.json')!;
  const { config } = ts.readConfigFile(configPath, ts.sys.readFile);
  const { options } = ts.parseJsonConfigFileContent(config, ts.sys, root);
  const entry = resolve(root, 'src/index.ts');
  const program = ts.createProgram([entry], { ...options, noEmit: true });
  const checker = program.getTypeChecker();
  const moduleSymbol = checker.getSymbolAtLocation(program.getSourceFile(entry)!)!;
  return checker.getExportsOfModule(moduleSymbol).map((symbol) => {
    const target = symbol.flags & ts.SymbolFlags.Alias ? checker.getAliasedSymbol(symbol) : symbol;
    return {
      name: symbol.name,
      isValue: (target.flags & ts.SymbolFlags.Value) !== 0,
      isDeprecated: target.getJsDocTags(checker).some((tag) => tag.name === 'deprecated'),
    };
  });
})();
const publicNames = new Set(publicExports.map((e) => e.name));

// Ids Storybook publishes: `<title>--docs` for autodocs metas and MDX pages, `<title>--<story>` per export.
const storybookIds = (() => {
  const ids = new Set<string>();
  const storyModules = import.meta.glob('./**/*.stories.{ts,tsx}', { eager: true }) as Record<
    string,
    { default: { title: string; tags?: readonly string[] } } & Record<string, unknown>
  >;
  Object.values(storyModules).forEach(({ default: meta, ...stories }) => {
    if (meta.tags?.includes('autodocs')) ids.add(`${sanitize(meta.title)}--docs`);
    Object.keys(stories).forEach((name) => ids.add(toId(meta.title, storyNameFromExport(name))));
  });
  const mdxPages = import.meta.glob('./**/*.mdx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>;
  const moduleKey = (file: string) => posix.normalize(file).replace(/\.tsx?$/, '');
  Object.entries(mdxPages).forEach(([file, mdx]) => {
    const title = mdx.match(/<Meta[^>]*\btitle="([^"]+)"/)?.[1];
    if (title) {
      ids.add(`${sanitize(title)}--docs`);
      return;
    }
    // `<Meta of={Stories} />` attaches the page to a stories module imported in the same file.
    const attached = mdx.match(/<Meta[^>]*\bof=\{(\w+)\}/)?.[1];
    const from = attached && mdx.match(new RegExp(`import \\* as ${attached} from '([^']+)'`))?.[1];
    if (!from) return;
    const target = moduleKey(`${posix.dirname(file)}/${from}`);
    const stories = Object.entries(storyModules).find(([key]) => moduleKey(key) === target)?.[1];
    if (stories) ids.add(`${sanitize(stories.default.title)}--docs`);
  });
  return ids;
})();

// PMM-15492: llms.txt ships in the tarball and indexes every export.
describe('llms.txt', () => {
  it('is listed in package.json files', () => {
    expect(pkg.files).toContain('llms.txt');
  });

  it('follows the llms.txt layout: H1, blockquote, then H2 sections of link items', () => {
    expect(lines[0]).toBe('# Peak UI');
    const firstSection = lines.findIndex((line) => line.startsWith('## '));
    expect(firstSection).toBeGreaterThan(0);
    expect(lines.slice(1, firstSection).some((line) => line.startsWith('> '))).toBe(true);
    lines.slice(firstSection).forEach((line, offset) => {
      if (line === '' || line.startsWith('## ')) return;
      expect(line, `line ${firstSection + offset + 1}`).toMatch(LINK_ITEM);
    });
  });

  it('discovers the public exports', () => {
    expect(publicExports.length).toBeGreaterThan(100);
  });

  it.each(publicExports.filter((e) => e.isValue).map((e) => e.name))(
    'has a line for %s',
    (name) => {
      expect(llms).toContain(`[${name}](`);
    }
  );

  it.each(publicExports.filter((e) => !e.isValue).map((e) => e.name))(
    'mentions the type %s',
    (name) => {
      expect(llms).toMatch(new RegExp(`(^|[^A-Za-z0-9_])${name}([^A-Za-z0-9_]|$)`));
    }
  );

  it('lists only names the entry point exports', () => {
    const notExports = new Set(['Guides', 'Not exported: use MUI', 'Optional']);
    let heading = '';
    const listed = lines.flatMap((line) => {
      if (line.startsWith('## ')) heading = line.slice(3);
      const name = line.match(/^- \[(\w+)\]\(/)?.[1];
      return name && !notExports.has(heading) ? [name] : [];
    });
    expect(listed.length).toBeGreaterThan(100);
    listed.forEach((name) => expect(publicNames.has(name), name).toBe(true));
  });

  it('discovers the Storybook pages', () => {
    expect(storybookIds.size).toBeGreaterThan(0);
  });

  it.each(links.filter((l) => l.startsWith(STORYBOOK)))(
    'links to a Storybook page that exists: %s',
    (link) => {
      const [kind, id] = link.slice(STORYBOOK.length).split('/');
      expect(['docs', 'story']).toContain(kind);
      if (kind === 'docs') expect(id).toMatch(/--docs$/);
      expect(storybookIds.has(id), id).toBe(true);
    }
  );

  it.each(links.filter((l) => l.startsWith(SOURCE)))(
    'links to a source file that exists: %s',
    (link) => {
      expect(existsSync(resolve(root, link.slice(SOURCE.length)))).toBe(true);
    }
  );
});

// Facts the prose repeats from code or from the other guides must match their source.
describe('llms.txt stays in sync', () => {
  it('finds the @deprecated exports', () => {
    expect(publicExports.filter((e) => e.isDeprecated).length).toBeGreaterThan(0);
  });

  it.each(publicExports.filter((e) => e.isValue).map((e) => [e.name, e.isDeprecated] as const))(
    'marks %s as deprecated only when its JSDoc says so (%s)',
    (name, isDeprecated) => {
      expect(/\): Deprecated\b/.test(lineFor(name))).toBe(isDeprecated);
    }
  );

  it('names only non-exports in the "Not exported: use MUI" section', () => {
    const named = section('Not exported: use MUI')
      .flatMap((line) => [...line.matchAll(/`([A-Z]\w+)`/g)].map((m) => m[1]))
      .concat(
        section('Not exported: use MUI').flatMap(
          (line) => line.match(/^- \[([^\]]+)\]/)?.[1].split(/,? and |, /) ?? []
        )
      );
    expect(named.length).toBeGreaterThan(0);
    named.forEach((name) => expect(publicNames.has(name), name).toBe(false));
  });

  it('quotes the real defaults', () => {
    expect(lineFor('PageContainer')).toContain(`${DEFAULT_PAGE_MAX_WIDTH}px`);
    expect(lineFor('DEFAULT_PAGE_MAX_WIDTH')).toContain(`(${DEFAULT_PAGE_MAX_WIDTH})`);
    const { pageIndex, pageSize } = DEFAULT_TABLE_STATE.pagination;
    expect(lineFor('DEFAULT_TABLE_STATE')).toContain(`page ${pageIndex} of ${pageSize} rows`);
  });

  it('lists the maturity tags exactly as Storybook labels them', () => {
    expect(llms).toContain(`(${MATURITY_TAGS.map((tag) => tag.label).join(', ')})`);
  });

  it('carries every URL AGENTS.md links to', () => {
    const agents = read('AGENTS.md');
    const urls = [...agents.slice(agents.indexOf('## Links')).matchAll(/https?:\/\/\S+/g)]
      .map((m) => m[0])
      .filter((url) => !url.endsWith('/llms.txt'));
    expect(urls.length).toBeGreaterThan(0);
    urls.forEach((url) => expect(llms, url).toContain(url));
  });
});
