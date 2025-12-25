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
	ErrInvalidSchedule             = "error_invalid_schedule"
	ErrInvalidIDFormat             = "error_invalid_id_format"
	ErrDatabaseError               = "error_database_error"
	ErrMissingParameter            = "error_missing_parameter"
	ErrInvalidRequestBody          = "error_invalid_request_body"
	ErrPathNotExist                = "error_path_not_exist"
	ErrPathNotDirectory            = "error_path_not_directory"
	ErrRemoteNotFound              = "error_remote_not_found"
	ErrJobNotActive                = "error_job_not_active"
	ErrJobNotFound                 = "error_job_not_found"
	ErrProviderNotFound            = "error_provider_not_found"
	ErrConnectionTestFailed        = "error_connection_test_failed"
	ErrFailedToListRemotes         = "error_failed_to_list_remotes"
	ErrFailedToCreateRemote        = "error_failed_to_create_remote"
	ErrFailedToGetQuota            = "error_failed_to_get_quota"
	ErrImportParseFailed           = "error_import_parse_failed"
	ErrImportEmptyList             = "error_import_empty_list"
	ErrConnectionHasDependentTasks = "error_connection_has_dependent_tasks"
)

// Status message keys
const (
	StatusSyncing      = "status_syncing"
	StatusSyncingFiles = "status_syncing_files"
	StatusCompleted    = "status_completed"
	StatusFailed       = "status_failed"
	StatusIdle         = "status_idle"
	StatusCancelled    = "status_cancelled"
)

// Success message keys
const (
	SuccessCreated   = "success_created"
	SuccessUpdated   = "success_updated"
	SuccessDeleted   = "success_deleted"
	SuccessSyncStart = "success_sync_started"
)
