import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import ts from 'typescript';
import { describe, expect, it } from 'vitest';

// PMM-15493: every prop or option of the public API carries a one-line JSDoc description, so
// editors and the built .d.ts explain what it does for the user. Keep the sentence in UX terms.
const root = join(process.cwd(), 'src');
const GUARDED = /(Props|Options)$/;
const SKIP = /\.(stories|spec)\.tsx?$|\.d\.ts$/;

const sourceFiles = readdirSync(root, { recursive: true })
  .map(String)
  .filter((file) => /\.tsx?$/.test(file) && !SKIP.test(file))
  .sort();

type Member = { file: string; type: string; prop: string };

const memberName = (member: ts.TypeElement) =>
  member.name && ts.isIdentifier(member.name) ? member.name.text : member.name?.getText();

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
  const path = join(root, file);
  const source = ts.createSourceFile(
    path,
    readFileSync(path, 'utf8'),
    ts.ScriptTarget.Latest,
    true
  );
  const checked: Member[] = [];
  const undocumented: Member[] = [];
  const visit = (type: string, members: readonly ts.TypeElement[]) => {
    members
      .filter((member) => ts.isPropertySignature(member) || ts.isMethodSignature(member))
      .forEach((member) => {
        const entry = {
          file: relative(process.cwd(), path),
          type,
          prop: memberName(member) ?? '?',
        };
        checked.push(entry);
        if (!hasDescription(member)) undocumented.push(entry);
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

describe('component props carry a JSDoc description', () => {
  const results = sourceFiles.map(collect);
  const checked = results.flatMap((r) => r.checked);
  const undocumented = results.flatMap((r) => r.undocumented);

  it('finds the props to check', () => {
    expect(sourceFiles.length).toBeGreaterThan(0);
    expect(checked.length).toBeGreaterThan(100);
    expect(checked.map((m) => m.type)).toContain('TableProps');
    expect(checked.map((m) => m.type)).toContain('TextInputProps');
  });

  it('describes every prop of every *Props and *Options type', () => {
    const report = undocumented.map((m) => `${m.file}: ${m.type}.${m.prop}`).join('\n');
    expect(undocumented, `Add a one-line /** description */ above:\n${report}`).toEqual([]);
  });
});
