# Native QAN — design lock (Figma)

Source: **PMM Use Cases With AI** (`PEujHp1KIneo0tbrXuDHX1`), frames **Desktop 3–6**, **13–14**.

Native QAN is a **successor UX**, not a Grafana QAN port. Grafana remains a legacy fallback while `nativeQanEnabled` is off.

## Primary journey

1. Land on **Query Analytics** with time range and optional filters applied.
2. Scan the **query listing** (sort, paginate, search dimensions).
3. Select a row → **Query Fingerprint** panel opens below the listing (~50/50 split).
4. Review fingerprint, examples, explain/plan, or tables in **section tabs**.
5. Use **Get AI Insights** (tab) or the **400px AI aside** for advisory chat (copy-only; no auto-apply).

Totals row (first listing row): selects aggregate view; section tabs hidden (same as Grafana semantics).

## Layout (Desktop 6 / 13 / 14)

```
┌─────────────────────────────────────────────────────────────────────────┐
│ PMM Header (app shell, 72px) — Historical / Real-Time QAN tabs          │
├─────────────────────────────────────────────────────────────────────────┤
│ Controls: [Filters]  chip… chip…  Clear all  │  time · group · search … │
├────────────────────────────── main (~1094px) ────────────┬── AI 400px ──┤
│ Listing (~46% height when query selected)                  │ Chat window  │
├──────────────────────────────────────────────────────────┤              │
│ Section Tab — “Query Fingerprint” (~46%)                   │ Chat input   │
│  Details | Examples | Explain Plan | Tables | ✨ Get AI …  │              │
└────────────────────────────────────────────────────────────┴──────────────┘
```

- **Filters aside (240px):** hidden by default; opened from **Filters** button (Desktop 3–5 show `Aside hidden="true"`).
- **AI aside:** always visible when ADRE is configured; 400px fixed width.
- **No left filter column** in the default view — filters live in a drawer + chip bar.

## Section tabs (replaces Grafana tab names)

| Figma label | Purpose | Internal panel |
|-------------|---------|----------------|
| **Details** | Fingerprint SQL + metrics summary; optional anomaly promo card | `QanDetailsOverview` |
| **Examples** | Query examples | `QanExamplesTab` |
| **Explain Plan** | MySQL EXPLAIN or PostgreSQL plan (single tab) | `QanExplainPlanTab` |
| **Tables** | Schema / tables | `QanTablesTab` |
| **Get AI Insights** | Batch Holmes analysis (advisory) | `QanAiInsightsTab` |

PostgreSQL: **Explain Plan** shows plan only (no classic explain tab). MongoDB: Explain Plan and Tables hidden.

Persistent **AI chat** stays in the right aside; **Get AI Insights** is the async analysis tab.

## URL state (unchanged)

Query params remain in `useQanPanelState` (`from`, `to`, `filter_*`, `query_id`, `tab`, etc.).

Tab aliases: `explain` and `plan` → **Explain Plan** section.

## Non-goals

- Grafana URL (`var-*`) compatibility
- Pixel-perfect Figma export (implement with MUI + `@percona/percona-ui`)
- Auto-apply SQL, IDE integration

## Acceptance (design, not Grafana parity)

- [ ] Three-zone layout: main + section tab + 400px AI aside
- [ ] Filters drawer + chip bar
- [ ] Query Fingerprint header with section tabs per Figma labels
- [ ] Advisory-only AI (aside + Get AI Insights)
- [ ] MySQL / PostgreSQL / MongoDB section visibility rules
- [ ] `nativeQanEnabled` off by default until product sign-off
