// Package model provides GraphQL model types and enum implementations.
package model

func toStrings[T ~string](arr []T) []string {
	strs := make([]string, len(arr))
	for i, v := range arr {
		strs[i] = string(v)
	}
	return strs
}

// Values returns all valid values for JobTrigger enum.
func (JobTrigger) Values() []string {
	return toStrings(AllJobTrigger)
}

// Values returns all valid values for JobStatus enum.
func (JobStatus) Values() []string {
	return toStrings(AllJobStatus)
}

// Values returns all valid values for LogLevel enum.
func (LogLevel) Values() []string {
	return toStrings(AllLogLevel)
}

// Values returns all valid values for LogAction enum.
func (LogAction) Values() []string {
	return toStrings(AllLogAction)
}

// Values returns all valid values for SyncDirection enum.
func (SyncDirection) Values() []string {
	return toStrings(AllSyncDirection)
}
