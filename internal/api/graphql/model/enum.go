package model

func toStrings[T ~string](arr []T) []string {
	strs := make([]string, len(arr))
	for i, v := range arr {
		strs[i] = string(v)
	}
	return strs
}

func (_ JobTrigger) Values() []string {
	return toStrings(AllJobTrigger)
}

func (_ JobStatus) Values() []string {
	return toStrings(AllJobStatus)
}

func (_ LogLevel) Values() []string {
	return toStrings(AllLogLevel)
}

func (_ LogAction) Values() []string {
	return toStrings(AllLogAction)
}

func (_ SyncDirection) Values() []string {
	return toStrings(AllSyncDirection)
}
