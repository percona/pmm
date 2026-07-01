// ---------------------------------------------------------------------------
// Per-node override editing for a threshold PromQL expression.
//
// Works with expressions of the form:
//
//   label_replace(vector(10), "node_name", "pmm-server", "", "")
//   or <anything>            <-- the default clause / metric can be ARBITRARY
//
// The functions only touch the `label_replace(...)` override clauses.
// The metric, the default threshold and its shape are left untouched.
// ---------------------------------------------------------------------------

const DEFAULT_LABEL = 'node_name';

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** Regex matching one node's override clause (the vector value is a group). */
function clauseRegExp(node: string, label: string): RegExp {
  return new RegExp(
    `label_replace\\(\\s*vector\\(\\s*([-0-9.]+)\\s*\\)\\s*,` +
      `\\s*"${escapeRegExp(label)}"\\s*,` +
      `\\s*"${escapeRegExp(node)}"\\s*,` +
      `\\s*""\\s*,\\s*""\\s*\\)`
  );
}

/**
 * Add or update a per-node override.
 *  - if the node already has an override -> replace its value in place
 *  - otherwise -> prepend a new override (front, so it wins the `or`)
 * Everything else in the expression is preserved verbatim.
 */
export function setOverride(
  expr: string,
  node: string,
  value: number,
  label = DEFAULT_LABEL
): string {
  const re = clauseRegExp(node, label);
  const newClause = `label_replace(vector(${value}), "${label}", "${node}", "", "")`;

  if (re.test(expr)) {
    return expr.replace(re, newClause); // update existing
  }
  return `${newClause}\nor ${expr}`; // add new (must precede the default)
}

/** Remove a node's override, dropping the adjacent `or` connector. */
export function removeOverride(
  expr: string,
  node: string,
  label = DEFAULT_LABEL
): string {
  const clause = clauseRegExp(node, label).source;

  // clause followed by "or"  (not the last term)
  const leading = new RegExp(`${clause}\\s*or\\s+`);
  if (leading.test(expr)) return expr.replace(leading, '');

  // clause preceded by "or"  (last term)
  return expr.replace(new RegExp(`\\s*or\\s+${clause}`), '');
}

// --- applying to the full Grafana alert-rule object -------------------------

interface AlertQuery {
  refId: string;
  model: { expr: string; [k: string]: unknown };
  [k: string]: unknown;
}
export interface AlertRule {
  data: AlertQuery[];
  [k: string]: unknown;
}

/**
 * Apply an expr transform to one query of an alert rule (default refId "B").
 * Returns a new rule object; the input is not mutated.
 */
export function updateAlertRuleExpr(
  rule: AlertRule,
  transform: (expr: string) => string,
  refId = 'B'
): AlertRule {
  const query = rule.data.find((d) => d.refId === refId);
  if (!query) throw new Error(`Query "${refId}" not found in rule.data`);

  const nextExpr = transform(query.model.expr);
  return {
    ...rule,
    data: rule.data.map((d) =>
      d.refId === refId ? { ...d, model: { ...d.model, expr: nextExpr } } : d
    ),
  };
}

// --- example usage ----------------------------------------------------------
// const rule: AlertRule = JSON.parse(rawJson);
//
// // update existing override (pmm-server -> 90)
// let updated = updateAlertRuleExpr(rule, (e) => setOverride(e, "pmm-server", 90));
//
// // add a new override (db-node-1 -> 60)
// updated = updateAlertRuleExpr(updated, (e) => setOverride(e, "db-node-1", 60));
//
// const outJson = JSON.stringify(updated, null, 4);
