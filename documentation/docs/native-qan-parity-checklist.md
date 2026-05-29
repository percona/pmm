# Native QAN parity checklist

Use this checklist before enabling `nativeQanEnabled` by default.

## Engines

- [ ] MySQL — overview, filters, all detail tabs
- [ ] PostgreSQL — overview, Plan tab
- [ ] MongoDB — overview, tabs hidden per engine rules

## Tabs (query selected, group by query)

- [ ] Details (metrics + metadata)
- [ ] Examples
- [ ] Explain (classic + JSON)
- [ ] Tables / schema
- [ ] AI Insights (advisory, copy-to-clipboard)
- [ ] Plan (PostgreSQL only)

## Navigation & integration

- [ ] Sidebar → `/pmm-ui/qan` when flag on; Grafana when off
- [ ] QanHeader Historical / Real-Time tabs
- [ ] Share link copy preserves filters and time range
- [ ] `pmm_ui_focus_qan_query` opens native QAN
- [ ] `/qan/ai-insights` redirects to native QAN when flag on
- [ ] Global ADRE widget hidden on `/qan`
- [ ] Embedded AI aside visible; advisory footer present

## Non-goals (confirm absent)

- [ ] No Apply / Run fix / migration buttons
- [ ] No IDE or terminal integration

## Performance

- [ ] Overview load acceptable on large services (10k+ queries in window)

Grafana QAN remains at `/pmm-ui/graph/d/pmm-qan/pmm-query-analytics` for comparison during QA.
