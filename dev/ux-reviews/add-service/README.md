# Add service UX review (PMM-15003)

Working files for the UX review of the Add service flow. They are kept on this branch so the
review can be reproduced; they are not meant to be merged.

- `index.html` and `img/`: the review as published (open `index.html` in a browser).
- `mockup-app/`: the React app that renders the mockups with `@percona/peak-ui` and MUI.
  Run `npm install --legacy-peer-deps && npx vite`, then open `/?s=<screen>&mode=dark|light`
  with `<screen>` one of `select`, `mysql`, `mysql-ok`, `mysql-fail`, `postgresql`, `rds-auth`,
  `rds`, `success`.
- `scripts/`: `playwright-cli run-code` scripts used for the live checks against a PMM Server
  (keyboard order, axe-core scans, viewports, data loss, invalid input, payload capture). They
  expect a browser session that is already logged in and point at `pmm.kubine.cc`.
