#!/usr/bin/env node
/**
 * Token Sync — reads a Figma variables export (variables.json) and patches
 * the colour-token declarations in BaseTheme.tsx and PmmTheme.tsx.
 *
 * Usage:  node scripts/tokensync/sync.mjs            (apply changes)
 *         node scripts/tokensync/sync.mjs --dry-run   (preview only)
 */

import { readFileSync, writeFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

const VARIABLES_PATH = resolve(__dirname, "variables.json");
const BASE_THEME_PATH = resolve(
  __dirname,
  "../../src/design/themes/base/BaseTheme.tsx",
);
const PMM_THEME_PATH = resolve(
  __dirname,
  "../../src/design/themes/pmm/PmmTheme.tsx",
);

const dryRun = process.argv.includes("--dry-run");
const variables = JSON.parse(readFileSync(VARIABLES_PATH, "utf8"));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getAtPath(obj, dotPath) {
  return dotPath.split(".").reduce((cur, key) => cur?.[key], obj);
}

function upperHex(s) {
  return s.replace(/#[0-9a-fA-F]+/g, (m) => m.toUpperCase());
}

/**
 * Recursively resolve a Figma value (which may be a reference or mode-keyed
 * object) down to either a primitives reference or a literal value.
 *
 * Returns { type: 'ref', path } | { type: 'number', value } | { type: 'literal', value }
 */
function resolveToCode(rawValue, mode) {
  if (rawValue && typeof rawValue === "object" && !Array.isArray(rawValue)) {
    if ("$value" in rawValue) return resolveToCode(rawValue.$value, mode);
    if (mode && mode in rawValue) return resolveToCode(rawValue[mode], mode);
  }
  if (
    typeof rawValue === "string" &&
    rawValue.startsWith("{") &&
    rawValue.endsWith("}")
  ) {
    const refPath = rawValue.slice(1, -1);
    if (refPath.startsWith("primitives.")) return { type: "ref", path: refPath };
    const resolved = getAtPath(variables, refPath);
    if (resolved != null) return resolveToCode(resolved, mode);
  }
  if (typeof rawValue === "number") return { type: "number", value: rawValue };
  return { type: "literal", value: String(rawValue) };
}

/** Turn a resolved code-value into a TypeScript expression string. */
function toTs(resolved) {
  if (resolved.type === "ref") {
    const parts = resolved.path.split(".");
    const last = parts.at(-1);
    return /^\d+$/.test(last)
      ? `${parts.slice(0, -1).join(".")}[${last}]`
      : parts.join(".");
  }
  if (resolved.type === "number") return String(resolved.value);
  return `"${upperHex(resolved.value)}"`;
}

// Shorthand: resolve a Figma node for a given mode straight to TS code.
const r = (node, mode) => toTs(resolveToCode(node, mode));

// ---------------------------------------------------------------------------
// Code generators
// ---------------------------------------------------------------------------

function buildPrimitivesTs(prims) {
  const lines = ["{"];

  const emitScale = (obj, indent) => {
    const keys = Object.keys(obj).sort((a, b) => Number(b) - Number(a));
    for (const key of keys) {
      lines.push(`${indent}${key}: "${upperHex(obj[key].$value)}",`);
    }
  };

  lines.push("  brand: {");
  for (const [name, scale] of Object.entries(prims.brand)) {
    lines.push(`    ${name}: {`);
    emitScale(scale, "      ");
    lines.push("    },");
  }
  lines.push("  },");

  lines.push("  extra: {");
  for (const [name, scale] of Object.entries(prims.extra)) {
    lines.push(`    ${name}: {`);
    emitScale(scale, "      ");
    lines.push("    },");
  }
  lines.push("  },");

  lines.push("  base: {");
  for (const [name, entry] of Object.entries(prims.base)) {
    lines.push(`    ${name}: "${upperHex(entry.$value)}",`);
  }
  lines.push("  }");

  lines.push("}");
  return lines.join("\n");
}

function buildSemanticTs(basis, mode) {
  const v = (node) => r(node, mode);
  const label = mode.toLowerCase();
  const lines = ["{"];

  lines.push(`  // Structural UI (${label} mode)`);
  lines.push("  surfaces: {");
  lines.push(`    elevation0: ${v(basis.surfaces.elevation0)},`);
  lines.push(`    elevation1: ${v(basis.surfaces.elevation1)},`);
  lines.push(`    backdrop: ${v(basis.surfaces.backdrop)},`);
  lines.push("  },");

  lines.push("  text: {");
  lines.push(`    primary: ${v(basis.text.primary)},`);
  lines.push(`    secondary: ${v(basis.text.secondary)},`);
  lines.push(`    disabled: ${v(basis.text.disabled)},`);
  lines.push(`    sky: ${v(basis.text.sky)},`);
  lines.push(`    lavender: ${v(basis.text.lavender)},`);
  lines.push(`    aqua: ${v(basis.text.aqua)},`);
  lines.push("  },");

  lines.push("  lines: {");
  lines.push(`    contour: ${v(basis.lines.contour)},`);
  lines.push(`    divider: ${v(basis.lines.divider)},`);
  lines.push(`    dividerStrong: ${v(basis.lines.dividerStrong)},`);
  lines.push(`    dividerStronger: ${v(basis.lines.dividerStronger)},`);
  lines.push("  },");

  lines.push("  action: {");
  lines.push(`    hover: ${v(basis.action.hover)},`);
  lines.push(`    disabled: ${v(basis.action.disabled)},`);
  lines.push(`    focus: ${v(basis.action.focus)},`);
  lines.push("  },");

  lines.push(`  // Semantic (${label} mode)`);

  for (const group of ["error", "success", "informative", "warning", "neutral"]) {
    const section = basis[group];
    lines.push(`  ${group}: {`);
    lines.push(`    light: ${v(section.light)},`);
    lines.push(`    main: ${v(section.main)},`);
    lines.push(`    dark: ${v(section.dark)},`);
    lines.push(`    surface: ${v(section.surface)},`);
    lines.push(`    contrastText: ${v(section.contrastText)},`);
    lines.push("  },");
  }

  lines.push(`  // Charts (${label} mode)`);
  lines.push("  charts: {");
  for (const key of ["stone", "sky", "aqua", "sunrise", "sunset", "red", "lavender"]) {
    lines.push(`    ${key}: ${v(basis.charts[key])},`);
  }
  lines.push("  }");

  lines.push("}");
  return lines.join("\n");
}

function buildPrimaryTs(section, mode) {
  const v = (node) => r(node, mode);
  return [
    "{",
    `  main: ${v(section.main)},`,
    `  dark: ${v(section.dark)},`,
    `  light: ${v(section.light)},`,
    `  contrastText: ${v(section.contrastText)},`,
    `  hover: ${v(section.hover)},`,
    `  selected: ${v(section.selected)},`,
    `  focus: ${v(section.focus)},`,
    `  focusVisible: ${v(section.focusVisible)},`,
    `  outlinedBorder: ${v(section.outlinedBorder)},`,
    "}",
  ].join("\n");
}

// ---------------------------------------------------------------------------
// Source-file replacement (brace-aware)
// ---------------------------------------------------------------------------

function replaceConstObject(source, name, newBody, asConst = false) {
  const prefix = `export const ${name} = `;
  const start = source.indexOf(prefix);
  if (start === -1) {
    console.warn(`  ⚠  Could not find "export const ${name}"`);
    return source;
  }

  const braceStart = source.indexOf("{", start + prefix.length);
  if (braceStart === -1) return source;

  let depth = 0;
  let braceEnd = -1;
  for (let i = braceStart; i < source.length; i++) {
    const ch = source[i];
    if (ch === '"' || ch === "'" || ch === "`") {
      i++;
      while (i < source.length && source[i] !== ch) {
        if (source[i] === "\\") i++;
        i++;
      }
      continue;
    }
    if (ch === "{") depth++;
    if (ch === "}") {
      depth--;
      if (depth === 0) {
        braceEnd = i;
        break;
      }
    }
  }
  if (braceEnd === -1) {
    console.warn(`  ⚠  Unbalanced braces for "${name}"`);
    return source;
  }

  let endIdx = braceEnd + 1;
  const tail = source.substring(endIdx);
  if (asConst && tail.startsWith(" as const;")) endIdx += " as const;".length;
  else if (tail.startsWith(";")) endIdx += 1;

  const suffix = asConst ? " as const;" : ";";
  return (
    source.substring(0, start) + prefix + newBody + suffix + source.substring(endIdx)
  );
}

// ---------------------------------------------------------------------------
// Generate all replacement bodies
// ---------------------------------------------------------------------------

const bodies = {
  primitives: buildPrimitivesTs(variables.primitives),
  semanticTokensLight: buildSemanticTs(variables.basis, "Light"),
  semanticTokensDark: buildSemanticTs(variables.basis, "Dark"),
  defaultPrimaryLight: buildPrimaryTs(variables.basis.primary.neutral, "Light"),
  defaultPrimaryDark: buildPrimaryTs(variables.basis.primary.neutral, "Dark"),
  pmmPrimaryLight: buildPrimaryTs(variables.basis.primary.pmm, "Light"),
  pmmPrimaryDark: buildPrimaryTs(variables.basis.primary.pmm, "Dark"),
};

// ---------------------------------------------------------------------------
// Apply to BaseTheme.tsx
// ---------------------------------------------------------------------------

let baseSource = readFileSync(BASE_THEME_PATH, "utf8");
baseSource = replaceConstObject(baseSource, "primitives", bodies.primitives, true);
baseSource = replaceConstObject(baseSource, "semanticTokensLight", bodies.semanticTokensLight);
baseSource = replaceConstObject(baseSource, "semanticTokensDark", bodies.semanticTokensDark);
baseSource = replaceConstObject(baseSource, "defaultPrimaryLight", bodies.defaultPrimaryLight);
baseSource = replaceConstObject(baseSource, "defaultPrimaryDark", bodies.defaultPrimaryDark);

if (dryRun) {
  console.log("--- BaseTheme.tsx (dry-run) ---\n");
  console.log(baseSource);
} else {
  writeFileSync(BASE_THEME_PATH, baseSource);
  console.log(`✓ Updated  ${BASE_THEME_PATH}`);
}

// ---------------------------------------------------------------------------
// Apply to PmmTheme.tsx
// ---------------------------------------------------------------------------

let pmmSource = readFileSync(PMM_THEME_PATH, "utf8");
pmmSource = replaceConstObject(pmmSource, "pmmPrimaryLight", bodies.pmmPrimaryLight);
pmmSource = replaceConstObject(pmmSource, "pmmPrimaryDark", bodies.pmmPrimaryDark);

if (dryRun) {
  console.log("\n--- PmmTheme.tsx (dry-run) ---\n");
  console.log(pmmSource);
} else {
  writeFileSync(PMM_THEME_PATH, pmmSource);
  console.log(`✓ Updated  ${PMM_THEME_PATH}`);
}

// ---------------------------------------------------------------------------
// Report unmatched Figma variables
// ---------------------------------------------------------------------------

console.log("\n━━━ Figma variables with no recipient in theme files ━━━\n");

const unmatched = [];

if (variables.basis?.surfaces?.elevationX) {
  const lightVal = resolveToCode(variables.basis.surfaces.elevationX, "Light");
  const darkVal = resolveToCode(variables.basis.surfaces.elevationX, "Dark");
  unmatched.push({
    path: "basis.surfaces.elevationX",
    light: toTs(lightVal),
    dark: toTs(darkVal),
  });
}

if (variables.tokens?.shape) {
  for (const [key, node] of Object.entries(variables.tokens.shape)) {
    const resolved = resolveToCode(node, null);
    unmatched.push({
      path: `tokens.shape.${key}`,
      value: toTs(resolved),
    });
  }
}

if (unmatched.length === 0) {
  console.log("  (none)");
} else {
  for (const entry of unmatched) {
    if (entry.light) {
      console.log(`  ${entry.path}  →  Light: ${entry.light}  |  Dark: ${entry.dark}`);
    } else {
      console.log(`  ${entry.path}  →  ${entry.value}`);
    }
  }
}

console.log();
