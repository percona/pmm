# Troubleshoot upgrade issues

## PMM Server not updating correctly

If the automatic update process isn't working, you can force an update using the API:

1. Open your terminal.
2. Run the update command, replacing <username>:<password> with your credentials and <pmm-server-address> with your PMM server addressЖ 
   
   ```curl -X POST \
   --user <username>:<password> \
   'http://<pmm-server-address>/v1/server/updates:start' \
   -H 'Content-Type: application/json'
   ```
3. Wait 2-5 minutes and refresh the PMM Home page to verify the update.