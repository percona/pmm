# Example Feature Request Ticket

```
Title: [Backup] Support scheduled backups for PostgreSQL via pg_dump

User Story
As a DBA managing PostgreSQL instances in PMM, I want to schedule automated backups
so that I can meet recovery point objectives without manual intervention.

Acceptance Criteria
- Backup > Scheduled Backups supports PostgreSQL as a target database type
- User can configure backup frequency (hourly, daily, weekly), retention policy, and storage location
- Backup status (success/failure) is visible in the Backup > All Backups list
- Failed backups trigger a configured PMM alert
- Restore from a PostgreSQL backup works end-to-end via the UI

Design / UI / UX
Extend the existing "Create Scheduled Backup" modal to include PostgreSQL in the
database type dropdown. No new screens needed. Attach mockup: backup-pg-modal.png.

Suggested Implementation
- Add pg_dump-based backup logic to pmm-agent (new backup artifact type)
- Extend the Backup gRPC API (pmm-managed) to accept PostgreSQL artifact metadata
- Reuse existing S3/local storage pipeline from MySQL backup implementation
- pmm-managed scheduler triggers via the existing cron-based job runner

Out of Scope
- Point-in-time recovery (PITR) using WAL archiving
- Backup encryption at rest
- Support for PostgreSQL replicas (primary only in this iteration)

Details
- Customer request tracked in Zendesk ticket #98234
- Related: PMM-11420 (MySQL scheduled backups — same pipeline to extend)
- PostgreSQL versions in scope: 13, 14, 15, 16

How to Document
Add a "PostgreSQL Backups" section to the existing Backup docs page. Add a release
note entry. Update the supported databases matrix on the Backup overview page.

How to test
1. Deploy PMM with a PostgreSQL 15 instance registered
2. Navigate to Backup > Scheduled Backups > Create
3. Select PostgreSQL, configure daily schedule, S3 storage
4. Verify backup job runs and artifact appears in All Backups with status "Success"
5. Restore the artifact and verify database integrity
6. Trigger a failure (revoke S3 permissions) and verify alert fires

Automatable
Yes, extend api-tests/backup suite.
```
