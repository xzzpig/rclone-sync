-- reverse: create index "task_created_at" to table: "tasks"
DROP INDEX `task_created_at`;
-- reverse: create index "task_connection_id" to table: "tasks"
DROP INDEX `task_connection_id`;
-- reverse: create "tasks" table
DROP TABLE `tasks`;
-- reverse: create index "joblog_job_id_time" to table: "job_logs"
DROP INDEX `joblog_job_id_time`;
-- reverse: create index "joblog_job_id" to table: "job_logs"
DROP INDEX `joblog_job_id`;
-- reverse: create "job_logs" table
DROP TABLE `job_logs`;
-- reverse: create index "job_status" to table: "jobs"
DROP INDEX `job_status`;
-- reverse: create index "job_task_id_start_time" to table: "jobs"
DROP INDEX `job_task_id_start_time`;
-- reverse: create index "job_task_id" to table: "jobs"
DROP INDEX `job_task_id`;
-- reverse: create "jobs" table
DROP TABLE `jobs`;
-- reverse: create index "connection_created_at" to table: "connections"
DROP INDEX `connection_created_at`;
-- reverse: create index "connection_type" to table: "connections"
DROP INDEX `connection_type`;
-- reverse: create index "connection_name" to table: "connections"
DROP INDEX `connection_name`;
-- reverse: create index "connections_name_key" to table: "connections"
DROP INDEX `connections_name_key`;
-- reverse: create "connections" table
DROP TABLE `connections`;
