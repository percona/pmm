# Example Bug Ticket

```
Title: [Alerting] Custom alerting rule silently dropped after PMM Server restart

Steps to reproduce
1. Deploy PMM 3.1.0 via Docker
2. Navigate to Alerting > Alert Rules > New Alert Rule
3. Create a custom rule with a PromQL expression
4. Restart the PMM Server container (`docker restart pmm-server`)
5. Navigate back to Alerting > Alert Rules

Actual result
The custom rule is no longer listed. No error in the UI. Server logs show:
`level=error msg="failed to restore rule" err="rule file not found"`

Expected result
Custom alerting rules persist across PMM Server restarts.

User impact
All users who create custom alerting rules lose them on restart. High severity — silent data loss of monitoring configuration.

Workaround
Export rules via API (`GET /v1/alerting/rules`) before restart and re-import after.

Details
- PMM 3.1.0, Docker deployment
- See attached pmm-managed.log and screenshot of empty rules page

How to document
Add to Known Issues for 3.1.0. Update "Alerting" docs page with a note about backup/export before restart.

How to test
1. Create 3 alert rules (1 built-in template, 1 custom PromQL, 1 with labels)
2. Restart PMM Server
3. Verify all 3 rules are present and functional

Automatable: yes, extend api-tests/alerting suite.
```
