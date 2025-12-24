-- create "connections" table
CREATE TABLE `connections` (
  `id` uuid NOT NULL,
  `name` text NOT NULL,
  `type` text NOT NULL,
  `encrypted_config` blob NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- create index "connections_name_key" to table: "connections"
CREATE UNIQUE INDEX `connections_name_key` ON `connections` (`name`);
-- create index "connection_name" to table: "connections"
CREATE UNIQUE INDEX `connection_name` ON `connections` (`name`);
-- create index "connection_type" to table: "connections"
CREATE INDEX `connection_type` ON `connections` (`type`);
-- create "jobs" table
CREATE TABLE `jobs` (
  `id` uuid NOT NULL,
  `status` text NOT NULL DEFAULT 'PENDING',
  `trigger` text NOT NULL,
  `start_time` datetime NOT NULL,
  `end_time` datetime NULL,
  `files_transferred` integer NOT NULL DEFAULT 0,
  `bytes_transferred` integer NOT NULL DEFAULT 0,
  `errors` text NULL,
  `task_id` uuid NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `jobs_tasks_jobs` FOREIGN KEY (`task_id`) REFERENCES `tasks` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create "job_logs" table
CREATE TABLE `job_logs` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `level` text NOT NULL,
  `time` datetime NOT NULL,
  `path` text NULL,
  `what` text NOT NULL DEFAULT 'UNKNOWN',
  `size` integer NULL,
  `job_id` uuid NOT NULL,
  CONSTRAINT `job_logs_jobs_logs` FOREIGN KEY (`job_id`) REFERENCES `jobs` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create "tasks" table
CREATE TABLE `tasks` (
  `id` uuid NOT NULL,
  `name` text NOT NULL,
  `source_path` text NOT NULL,
  `remote_path` text NOT NULL,
  `direction` text NOT NULL DEFAULT 'BIDIRECTIONAL',
  `schedule` text NULL,
  `realtime` bool NOT NULL DEFAULT false,
  `options` json NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `connection_id` uuid NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `tasks_connections_tasks` FOREIGN KEY (`connection_id`) REFERENCES `connections` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
