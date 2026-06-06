ALTER TABLE providers DROP COLUMN timeout_ms;
ALTER TABLE providers DROP COLUMN header_timeout_ms;
ALTER TABLE providers DROP COLUMN chunk_timeout_ms;
ALTER TABLE providers DROP COLUMN enterprise_url;
ALTER TABLE providers DROP COLUMN set_cache_key;
ALTER TABLE providers DROP COLUMN api_package;
ALTER TABLE providers DROP COLUMN env_list;

ALTER TABLE models DROP COLUMN interleaved;
ALTER TABLE models DROP COLUMN fine_tuning;
ALTER TABLE models DROP COLUMN classification;
ALTER TABLE models DROP COLUMN moderation;
ALTER TABLE models DROP COLUMN created_timestamp;
ALTER TABLE models DROP COLUMN owned_by;
