-- create "task_hooks" table
CREATE TABLE `task_hooks` (
  `id` uuid NOT NULL,
  `enabled` bool NOT NULL DEFAULT true,
  `priority` integer NULL,
  `event` text NOT NULL,
  `type` text NOT NULL,
  `on_error` text NOT NULL DEFAULT 'IGNORE',
  `config` json NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `connection_id` uuid NULL,
  `task_id` uuid NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `task_hooks_tasks_hooks` FOREIGN KEY (`task_id`) REFERENCES `tasks` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT `task_hooks_connections_hooks` FOREIGN KEY (`connection_id`) REFERENCES `connections` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "taskhook_task_id" to table: "task_hooks"
CREATE INDEX `taskhook_task_id` ON `task_hooks` (`task_id`);
-- create index "taskhook_connection_id" to table: "task_hooks"
CREATE INDEX `taskhook_connection_id` ON `task_hooks` (`connection_id`);
-- create index "taskhook_task_id_event_priority_created_at" to table: "task_hooks"
CREATE INDEX `taskhook_task_id_event_priority_created_at` ON `task_hooks` (`task_id`, `event`, `priority`, `created_at`);
-- create index "taskhook_connection_id_event_priority_created_at" to table: "task_hooks"
CREATE INDEX `taskhook_connection_id_event_priority_created_at` ON `task_hooks` (`connection_id`, `event`, `priority`, `created_at`);
-- create index "taskhook_created_at" to table: "task_hooks"
CREATE INDEX `taskhook_created_at` ON `task_hooks` (`created_at`);
