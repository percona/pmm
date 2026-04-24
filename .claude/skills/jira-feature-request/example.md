# Example Feature Request

```
Title: [Backup] Support retention policies for scheduled MongoDB backups

User Story
As a DBA managing scheduled MongoDB backups in PMM, I want to define a retention
policy per schedule, so that old backup artifacts are pruned automatically
without relying on out-of-band S3 lifecycle rules or manual cleanup.

Acceptance criteria
- A "Retention" section is available in the scheduled backup form with two
  mutually compatible fields: `keep_last` (integer) and `keep_for` (duration).
- The `CreateScheduledBackup` and `ChangeScheduledBackup` APIs accept the same
  retention fields; validation rejects `keep_last < 1` and malformed durations.
- After a successful backup run, pmm-managed prunes artifacts that violate the
  policy. Failed/in-progress backups are never pruned.
- If both `keep_last` and `keep_for` are set, an artifact is kept when either
  rule says to keep it (union, not intersection).
- The schedule exposes a `LastPruneAt` timestamp and emits pmm-managed log
  entries listing each pruned artifact.
- Existing schedules continue to work with no retention policy (unlimited).
- Removing a schedule does not prune existing artifacts.

Design / UI / UX (if applicable)
- New "Retention" fieldset between "Schedule" and "Advanced" in the backup form
  (see mockup: retention-field-mockup.png).
- Empty state: "No retention policy — backups are kept indefinitely."
- Validation errors appear inline below each input.
- Backup list view adds a small badge ("Retention: keep 7") next to schedules
  that have a policy set.

Suggested implementation / options
A. Extend the existing `backup_schedules` table with `keep_last` and `keep_for`
   columns; prune inside the existing post-run hook in pmm-managed. Simple,
   reuses existing transaction boundary, but mixes pruning with scheduling.
B. New `backup_retention_policies` table with a FK from `backup_schedules`.
   Cleaner separation, easier to extend later (e.g. GFS policies), at the cost
   of an extra join and migration.
Preference: (A) for the initial implementation, with a note to revisit if
grandfather-father-son or cross-schedule policies land.

Out of scope
- Retention for on-demand (non-scheduled) backups — tracked in PMM-XXXX.
- Postgres and MySQL scheduled-backup retention — follow-up once the data-model
  choice above is settled.
- S3 storage-class transitions (Glacier, etc.) — belongs in a separate ticket.
- UI for previewing which artifacts would be pruned before saving the policy.

Details
- Related: GitHub discussion #4821, Slack thread in #pmm-backup, support
  tickets SUP-1421, SUP-1502, SUP-1610.
- Prior art: pgBackRest `repo-retention-full`, Percona XtraBackup `--keep`.
- Affected components: pmm-managed (backup service), API (backup.proto), UI
  (Backup > Scheduled).
- Dependencies: none on other teams; requires a DB migration.
- Docs: update the "Scheduled Backups" page and add a release note.

Classification: New feature — targeting 3.2.0
```
