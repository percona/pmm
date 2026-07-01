# QAN AI Insights

**QAN AI Insights** provides **query optimisation and tuning** guidance based on **Query Analytics (QAN)** data for a specific query pattern. It is focused on performance analysis (plans, indexes, rewrites) rather than general infrastructure chat.

Use [**ADRE Chat**](adre-chat.md) for open-ended questions; use **QAN AI Insights** when you are already in **QAN** and want AI help on **that query**.

## Behaviour (conceptual)

- PMM sends QAN-related context to the configured **HolmesGPT** backend and returns an analysis.
- The server may **cache** the latest analysis per **query identifier** and **service** so repeated views do not always re-run a full analysis. Cached content has a timestamp; your UI may offer a control to refresh.

## Settings

- Administrators can adjust the **QAN insights prompt** (within size limits enforced by PMM) so the model follows your organisation’s preferred format or policies.
- **ADRE** must be enabled and **HolmesGPT** URL configured for insights to work.

## Privacy

Query text and metrics sent for analysis may contain **schema, SQL, and application identifiers**. Ensure HolmesGPT and network paths meet your compliance requirements.

API details for operators: see the **QAN insights** rows in the ADRE API table in [dev/adre/README.md](https://github.com/percona/pmm/blob/v3/dev/adre/README.md).

[← AI features overview](index.md)
