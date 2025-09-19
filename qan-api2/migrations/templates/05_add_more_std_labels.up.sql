ALTER TABLE metrics
  ADD COLUMN `node_id` LowCardinality(String) COMMENT 'Node identifier' AFTER `node_model`,
  ADD COLUMN `node_name` LowCardinality(String) COMMENT 'Node name' AFTER `node_id`,
  ADD COLUMN `node_type` LowCardinality(String) COMMENT 'Node type' AFTER `node_name`,
  ADD COLUMN `machine_id` LowCardinality(String) COMMENT 'Machine identifier' AFTER `node_type`,
  ADD COLUMN `container_id` LowCardinality(String) COMMENT 'Container identifier' AFTER `container_name`,
  ADD COLUMN `service_id` LowCardinality(String) COMMENT 'Service identifier' AFTER `service_type`
;
