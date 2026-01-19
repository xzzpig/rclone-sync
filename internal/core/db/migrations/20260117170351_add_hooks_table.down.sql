-- reverse: create index "taskhook_created_at" to table: "task_hooks"
DROP INDEX `taskhook_created_at`;
-- reverse: create index "taskhook_connection_id_event_priority_created_at" to table: "task_hooks"
DROP INDEX `taskhook_connection_id_event_priority_created_at`;
-- reverse: create index "taskhook_task_id_event_priority_created_at" to table: "task_hooks"
DROP INDEX `taskhook_task_id_event_priority_created_at`;
-- reverse: create index "taskhook_connection_id" to table: "task_hooks"
DROP INDEX `taskhook_connection_id`;
-- reverse: create index "taskhook_task_id" to table: "task_hooks"
DROP INDEX `taskhook_task_id`;
-- reverse: create "task_hooks" table
DROP TABLE `task_hooks`;
