// Package rclone provides rclone-related functions.
package rclone

import (
	"github.com/rclone/rclone/fs/filter"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

// createFilterFromRules creates a new rclone filter from a list of filter rules.
// Each rule should be in the format "- pattern" (exclude) or "+ pattern" (include).
// Returns the created filter or an error if any rule is invalid.
//
// Example valid rules:
//   - "- node_modules/**" (exclude node_modules directory)
//   - "+ *.jpg" (include all jpg files)
//   - "- *" (exclude all files)
//   - "+ **" (include all files recursively)
//
// This is a helper function used by ValidateFilterRules and other functions that need
// to create filters from rules. It eliminates code duplication across the codebase.
func createFilterFromRules(rules []string) (*filter.Filter, error) {
	if len(rules) == 0 {
		return nil, nil
	}

	fi, err := filter.NewFilter(nil)
	if err != nil {
		return nil, err
	}

	for i, rule := range rules {
		if err := fi.AddRule(rule); err != nil {
			return nil, i18n.NewI18nErrorWithData(i18n.ErrFilterRuleInvalid, map[string]interface{}{
				"Index":  i + 1,
				"Rule":   rule,
				"Reason": err.Error(),
			}).WithCause(err)
		}
	}

	return fi, nil
}

// ValidateFilterRules validates a list of rclone filter rules.
// Each rule should be in the format "- pattern" (exclude) or "+ pattern" (include).
// Returns nil if all rules are valid, otherwise returns an error with the first invalid rule.
//
// Example valid rules:
//   - "- node_modules/**" (exclude node_modules directory)
//   - "+ *.jpg" (include all jpg files)
//   - "- *" (exclude all files)
//   - "+ **" (include all files recursively)
func ValidateFilterRules(rules []string) error {
	if len(rules) == 0 {
		return nil
	}

	_, err := createFilterFromRules(rules)
	return err
}
