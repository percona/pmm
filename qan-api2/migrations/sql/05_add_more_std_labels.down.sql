ALTER TABLE metrics
  DROP COLUMN `node_id`,
  DROP COLUMN `node_name`,
  DROP COLUMN `node_type`,
  DROP COLUMN `machine_id`,
  DROP COLUMN `container_id`,
  DROP COLUMN `service_id`
;
