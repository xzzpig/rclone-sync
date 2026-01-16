-- add column "enabled" to table: "tasks"
ALTER TABLE `tasks` ADD COLUMN `enabled` bool NOT NULL DEFAULT true;
