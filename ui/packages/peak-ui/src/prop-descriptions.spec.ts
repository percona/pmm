import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import ts from 'typescript';
import { describe, expect, it } from 'vitest';

// PMM-15493: the public API describes every prop once, in its JSDoc; stories must not repeat it.
const root = join(process.cwd(), 'src');
const GUARDED = /(Props|Options)$/;
const SKIP = /\.(stories|spec)\.tsx?$|\.d\.ts$/;

const allFiles = readdirSync(root, { recursive: true }).map(String).sort();
const sourceFiles = allFiles.filter((file) => /\.tsx?$/.test(file) && !SKIP.test(file));
const storyFiles = allFiles.filter((file) => /\.stories\.tsx?$/.test(file));

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
  if (ts.isIntersectionTypeNode(node)) return node.types.flatMap(literalsOf);
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
    if (ts.isInterfaceDeclaration(statement) && GUARDED.test(statement.name.text)) {
      visit(statement.name.text, statement.members);
    }
    if (ts.isTypeAliasDeclaration(statement) && GUARDED.test(statement.name.text)) {
      literalsOf(statement.type).forEach((literal) => visit(statement.name.text, literal.members));
    }
  });
  return { checked, undocumented };
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
  });

  it('describes every prop of every *Props and *Options type', () => {
    const report = undocumented.map(location).join('\n');
    expect(undocumented, `Add a one-line /** description */ above:\n${report}`).toEqual([]);
  });
});

describe('stories leave the description to the JSDoc', () => {
  const duplicated = storyFiles.flatMap((story) => {
    const siblings = checked.filter((m) => dirname(m.file) === dirname(story));
    return describedArgTypes(story)
      .map((prop) => siblings.find((m) => m.prop === prop))
      .filter((m): m is Member => !!m)
      .map(
        (m) =>
          `${relative(process.cwd(), join(root, story))}: argTypes.${m.prop} repeats ${m.type}.${m.prop}`
      );
  });

  it('finds the stories to check', () => {
    expect(storyFiles.length).toBeGreaterThan(0);
    expect(storyFiles.some((story) => describedArgTypes(story).length > 0)).toBe(true);
  });

  it('never sets argTypes.<prop>.description for a prop the component declares', () => {
    expect(
      duplicated,
      `Delete the description; the JSDoc on the prop is shown instead:\n${duplicated.join('\n')}`
    ).toEqual([]);
  });
});
