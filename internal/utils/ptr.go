package utils

// ComparePtr compares two pointers of the same type.
// It returns true if both are nil, or if both are non-nil and their values are equal.
func ComparePtr[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
