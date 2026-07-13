# Native QAN acceptance criteria

Superseded framing: use [native-qan-design-lock.md](./native-qan-design-lock.md) as the product north star before enabling `nativeQanEnabled` by default.

Grafana QAN remains at `/pmm-ui/graph/d/pmm-qan/pmm-query-analytics` for comparison during QA.

## Engines

- [ ] MySQL — listing, filters, section tabs
- [ ] PostgreSQL — Explain Plan tab (plan)
- [ ] MongoDB — Explain Plan and Tables hidden

## Section tabs (query selected, group by query)

- [ ] Details (fingerprint + metrics + anomaly promo)
- [ ] Examples
- [ ] Explain Plan (MySQL EXPLAIN or PostgreSQL plan)
- [ ] Tables / schema
- [ ] Get AI Insights (advisory, copy-to-clipboard)

## Layout & integration

- [ ] Filters drawer (240px) + chip bar + Clear all
- [ ] Main + Query Fingerprint split + 400px AI aside
- [ ] Sidebar → `/pmm-ui/qan` when flag on; Grafana when off
- [ ] QanHeader Historical / Real-Time tabs
- [ ] Share link copy preserves filters and time range
- [ ] `pmm_ui_focus_qan_query` opens native QAN
- [ ] Global ADRE widget hidden on `/qan`
- [ ] Embedded AI aside visible; advisory footer present

## Non-goals (confirm absent)

- [ ] No Apply / Run fix / migration buttons
- [ ] No IDE or terminal integration
- [ ] No Grafana URL (`var-*`) compatibility requirement

## Performance

- [ ] Overview load acceptable on large services (10k+ queries in window)
