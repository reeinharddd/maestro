ALTER TABLE routing_rules DROP COLUMN priority_weight;
ALTER TABLE routing_rules DROP COLUMN enabled;
ALTER TABLE routing_rules DROP COLUMN created_at;
ALTER TABLE routing_rules DROP COLUMN updated_at;

DROP TABLE IF EXISTS model_budgets;
