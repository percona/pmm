import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import ts from 'typescript';
import { describe, expect, it } from 'vitest';

// PMM-15493: the public API describes every prop once, in its JSDoc; stories must not repeat it.
const root = join(process.cwd(), 'src');
const GUARDED = /(Props|Options)$/;

const SKIP = /\.(stories|spec)\.tsx?$|\.d\.ts$/;
const allFiles = readdirSync(root, { recursive: true }).map(String).sort();
const sourceFiles = allFiles.filter((file) => /\.tsx?$/.test(file) && !SKIP.test(file));
const storyFiles = allFiles.filter((file) => /\.stories\.tsx?$/.test(file));

const program = (() => {
  const cwd = process.cwd();
  const configPath = ts.findConfigFile(cwd, ts.sys.fileExists, 'tsconfig.json')!;
  const { config } = ts.readConfigFile(configPath, ts.sys.readFile);
  const { options } = ts.parseJsonConfigFileContent(config, ts.sys, cwd);
  const roots = [join(root, 'index.ts'), ...storyFiles.map((file) => join(root, file))];
  return ts.createProgram(roots, { ...options, noEmit: true });
})();
const checker = program.getTypeChecker();
const isOwnSource = (node: ts.Node) => {
  const fileName = node.getSourceFile().fileName;
  return fileName.startsWith(root) && !SKIP.test(fileName);
};

// Every type the entry point exports from this repo, so `*Result` and state types are checked too.
const publicTypes = (() => {
  const entry = program.getSourceFile(join(root, 'index.ts'))!;
  const moduleSymbol = checker.getSymbolAtLocation(entry)!;
  return new Set(
    checker
      .getExportsOfModule(moduleSymbol)
      .map((s) => (s.flags & ts.SymbolFlags.Alias ? checker.getAliasedSymbol(s) : s))
      .filter((s) => !(s.flags & ts.SymbolFlags.Value))
      .flatMap((s) => s.declarations ?? [])
      .filter((d) => d.getSourceFile().fileName.startsWith(root))
      .flatMap((d) =>
        ts.isInterfaceDeclaration(d) || ts.isTypeAliasDeclaration(d) ? [d.name.text] : []
      )
  );
})();
const guarded = (name: string) => GUARDED.test(name) || publicTypes.has(name);

type Member = { file: string; type: string; prop: string };

const parse = (file: string) => {
  const path = join(root, file);
  return ts.createSourceFile(path, readFileSync(path, 'utf8'), ts.ScriptTarget.Latest, true);
};

const nameOf = (node: ts.PropertyAssignment | ts.TypeElement) =>
  node.name && (ts.isIdentifier(node.name) || ts.isStringLiteral(node.name))
    ? node.name.text
    : node.name?.getText();

const hasDescription = (member: ts.TypeElement) =>
  ts
    .getJSDocCommentsAndTags(member)
    .some((doc) => ts.isJSDoc(doc) && !!ts.getTextOfJSDocComment(doc.comment)?.trim());

const literalsOf = (node: ts.TypeNode): ts.TypeLiteralNode[] => {
  if (ts.isTypeLiteralNode(node)) return [node];
  if (ts.isIntersectionTypeNode(node) || ts.isUnionTypeNode(node)) {
    return node.types.flatMap(literalsOf);
  }
  if (ts.isParenthesizedTypeNode(node)) return literalsOf(node.type);
  return [];
};

const collect = (file: string) => {
  const source = parse(file);
  const checked: Member[] = [];
  const undocumented: Member[] = [];
  const visit = (type: string, members: readonly ts.TypeElement[]) => {
    members
      .filter((member) => ts.isPropertySignature(member) || ts.isMethodSignature(member))
      .forEach((member) => {
        const entry = { file, type, prop: nameOf(member) ?? '?' };
        checked.push(entry);
        if (!hasDescription(member)) undocumented.push(entry);
        // nested object props (`state: { sorting: ... }`) are described too
        if (ts.isPropertySignature(member) && member.type) {
          literalsOf(member.type).forEach((literal) =>
            visit(`${type}.${entry.prop}`, literal.members)
          );
        }
      });
  };
  source.statements.forEach((statement) => {
    if (ts.isInterfaceDeclaration(statement) && guarded(statement.name.text)) {
      visit(statement.name.text, statement.members);
    }
    if (ts.isTypeAliasDeclaration(statement) && guarded(statement.name.text)) {
      literalsOf(statement.type).forEach((literal) => visit(statement.name.text, literal.members));
    }
  });
  return { checked, undocumented };
};

const unwrap = (node: ts.Expression): ts.Expression =>
  ts.isSatisfiesExpression(node) || ts.isAsExpression(node) || ts.isParenthesizedExpression(node)
    ? unwrap(node.expression)
    : node;

// The `component` the story's default-exported meta passes, resolved to its props type.
const storyComponentProps = (file: string) => {
  const source = program.getSourceFile(join(root, file))!;
  const exported = source.statements.find(ts.isExportAssignment)?.expression;
  if (!exported) return { found: false as const };
  let meta: ts.Expression = unwrap(exported);
  if (ts.isIdentifier(meta)) {
    const declaration = checker.getSymbolAtLocation(meta)?.valueDeclaration;
    if (declaration && ts.isVariableDeclaration(declaration) && declaration.initializer) {
      meta = unwrap(declaration.initializer);
    }
  }
  if (!ts.isObjectLiteralExpression(meta)) return { found: false as const };
  const component = meta.properties.find(
    (p): p is ts.PropertyAssignment => ts.isPropertyAssignment(p) && nameOf(p) === 'component'
  );
  if (!component) return { found: false as const };
  const type = checker.getTypeAtLocation(component.initializer);
  const param = (type.isUnion() ? type.types : [type])
    .flatMap((t) => [...t.getCallSignatures(), ...t.getConstructSignatures()])
    .map((sig) => sig.getParameters()[0])
    .find(Boolean);
  if (!param) return { found: true as const, props: undefined };
  const propsType = checker.getTypeOfSymbolAtLocation(param, component.initializer);
  const props = propsType.getProperties().flatMap((prop) =>
    (prop.declarations ?? []).filter(isOwnSource).map((d) => {
      let owner: ts.Node | undefined = d.parent;
      while (owner && !ts.isInterfaceDeclaration(owner) && !ts.isTypeAliasDeclaration(owner)) {
        owner = owner.parent;
      }
      const type = owner && (ts.isInterfaceDeclaration(owner) || ts.isTypeAliasDeclaration(owner));
      return { prop: prop.name, type: type ? owner.name.text : '?' };
    })
  );
  return { found: true as const, props };
};

// Names of the argTypes entries that set a `description`, anywhere in the story file.
const describedArgTypes = (file: string) => {
  const found: string[] = [];
  const visit = (node: ts.Node) => {
    if (
      ts.isPropertyAssignment(node) &&
      nameOf(node) === 'argTypes' &&
      ts.isObjectLiteralExpression(node.initializer)
    ) {
      node.initializer.properties.forEach((arg) => {
        if (
          ts.isPropertyAssignment(arg) &&
          ts.isObjectLiteralExpression(arg.initializer) &&
          arg.initializer.properties.some(
            (p) => ts.isPropertyAssignment(p) && nameOf(p) === 'description'
          )
        ) {
          found.push(nameOf(arg) ?? '?');
        }
      });
    }
    ts.forEachChild(node, visit);
  };
  visit(parse(file));
  return found;
};

const results = sourceFiles.map(collect);
const checked = results.flatMap((r) => r.checked);
const undocumented = results.flatMap((r) => r.undocumented);
const location = (m: Member) =>
  `${relative(process.cwd(), join(root, m.file))}: ${m.type}.${m.prop}`;

describe('component props carry a JSDoc description', () => {
  it('finds the props to check', () => {
    expect(sourceFiles.length).toBeGreaterThan(0);
    expect(checked.length).toBeGreaterThan(100);
    expect(checked.map((m) => m.type)).toContain('TableProps');
    expect(checked.map((m) => m.type)).toContain('TextInputProps');
    expect(checked.map((m) => m.type)).toContain('UseNavigableRowsResult');
  });

  it('describes every prop of every *Props and *Options type', () => {
    const report = undocumented.map(location).join('\n');
    expect(undocumented, `Add a one-line /** description */ above:\n${report}`).toEqual([]);
  });
});

describe('stories leave the description to the JSDoc', () => {
  const resolved = storyFiles.map((story) => ({ story, ...storyComponentProps(story) }));
  const where = (story: string) => relative(process.cwd(), join(root, story));
  const duplicated = resolved.flatMap(({ story, found, props }) =>
    !found || !props
      ? []
      : describedArgTypes(story)
          .map((name) => props.find((p) => p.prop === name))
          .filter((p): p is { prop: string; type: string } => !!p)
          .map((p) => `${where(story)}: argTypes.${p.prop} repeats ${p.type}.${p.prop}`)
  );

  it('resolves the props of every story component', () => {
    const unresolved = resolved.filter((r) => r.found && !r.props).map((r) => where(r.story));
    expect(
      unresolved,
      `Cannot read the props of the meta \`component\` in:\n${unresolved.join('\n')}`
    ).toEqual([]);
    const props = (story: string) => resolved.find((r) => r.story.endsWith(story))?.props ?? [];
    expect(props('code-block.stories.tsx').map((p) => p.type)).toContain('CodeBlockProps');
    expect(props('table.stories.tsx').map((p) => p.type)).toContain('TableProps');
  });

  it('never sets argTypes.<prop>.description for a prop the component declares', () => {
    expect(
      duplicated,
      `Delete the description; the JSDoc on the prop is shown instead:\n${duplicated.join('\n')}`
    ).toEqual([]);
  });
});
