# Update of PMM Server

Update of PMM Server which includes `managed` and other components is triggered by sending a [StartUpdate](https://github.com/percona/pmm/blob/6761010b8b30042936c58c022752f6b57581afee/api/serverpb/server.proto#L325) message.
This performs the following actions:
1. Runs [pmm-update](https://github.com/percona/pmm-update) command to initiate an update
2. `pmm-update` first updates itself to the latest version and restarts
3. `pmm-update` then runs a set of Ansible tasks to update all other components

## Notes
- `pmm-update` does not handle rollbacks in case of errors
- `pmm-update` requires root priveleges to run
