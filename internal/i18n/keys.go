package i18n

// Error message keys
const (
	ErrGeneric                     = "error_generic"
	ErrNotFound                    = "error_not_found"
	ErrAlreadyExists               = "error_already_exists"
	ErrUnauthorized                = "error_unauthorized"
	ErrValidationFailed            = "error_validation_failed"
	ErrConnectionFailed            = "error_connection_failed"
	ErrSyncFailed                  = "error_sync_failed"
	ErrTaskNotFound                = "error_task_not_found"
	ErrConnectionNotFound          = "error_connection_not_found"
	ErrInvalidInput                = "error_invalid_input"
	ErrDatabaseError               = "error_database_error"
	ErrMissingParameter            = "error_missing_parameter"
	ErrPathNotExist                = "error_path_not_exist"
	ErrProviderNotFound            = "error_provider_not_found"
	ErrConnectionTestFailed        = "error_connection_test_failed"
	ErrFailedToListRemotes         = "error_failed_to_list_remotes"
	ErrConnectionHasDependentTasks = "error_connection_has_dependent_tasks"
	ErrFilterRuleInvalid           = "error_filter_rule_invalid"
	ErrInvalidInfoAge              = "error_invalid_info_age"
	ErrInvalidChangeNotifyPoll     = "error_invalid_change_notify_poll"
	ErrChangeNotifyPollTooShort    = "error_change_notify_poll_too_short"
	ErrHookCancelled               = "error_hook_cancelled"
	ErrHookFatal                   = "error_hook_fatal"
	ErrHookTimeout                 = "error_hook_timeout"
	ErrHookHTTPFailed              = "error_hook_http_failed"
	ErrHookCommandFailed           = "error_hook_command_failed"
	ErrHookInvalidURL              = "error_hook_invalid_url"
	ErrHookInvalidTemplate         = "error_hook_invalid_template"
)

// Status message keys
const (
	StatusSyncingFiles = "status_syncing_files"
	StatusCompleted    = "status_completed"
	StatusFailed       = "status_failed"
	StatusIdle         = "status_idle"
	StatusCancelled    = "status_cancelled"
)

// Success message keys
const (
	SuccessCreated = "success_created"
)

// Cache message keys
const (
	CacheClearSuccess      = "cache_clear_success"
	CacheClearSuccessEmpty = "cache_clear_success_empty"
	CacheClearFailed       = "cache_clear_failed"
	CacheNotEnabled        = "cache_not_enabled"
)
