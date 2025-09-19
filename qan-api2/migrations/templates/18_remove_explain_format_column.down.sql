ALTER TABLE metrics
  ADD COLUMN `example_format` Enum8(
                                   'EXAMPLE_FORMAT_INVALID' = 0,
                                   'EXAMPLE' = 1,
                                   'FINGERPRINT' = 2
                                 ) COMMENT 'Indicates that collect real query examples is prohibited';
