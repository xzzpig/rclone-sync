// Package model provides GraphQL model types and validation functions.
package model

import (
	"time"

	"github.com/xzzpig/rclone-sync/internal/i18n"
)

// MinChangeNotifyPoll is the minimum allowed polling interval for ChangeNotify.
const MinChangeNotifyPoll = 10 * time.Second

// ValidateCacheOptions validates the cache options configuration.
// It returns an error if:
// - InfoAge is non-empty but not a valid Go duration format
// - ChangeNotifyPoll is non-empty but not a valid Go duration format
// - ChangeNotifyPoll is less than 10 seconds (MinChangeNotifyPoll)
func ValidateCacheOptions(opts *ConnectionCacheOptionsInput) error {
	if opts == nil {
		return nil
	}

	// Validate InfoAge
	if opts.InfoAge != nil && *opts.InfoAge != "" {
		if _, err := time.ParseDuration(*opts.InfoAge); err != nil {
			return i18n.NewI18nErrorWithData(i18n.ErrInvalidInfoAge, map[string]interface{}{"Error": err.Error()})
		}
	}

	// Validate ChangeNotifyPoll
	if opts.ChangeNotifyPoll != nil && *opts.ChangeNotifyPoll != "" {
		d, err := time.ParseDuration(*opts.ChangeNotifyPoll)
		if err != nil {
			return i18n.NewI18nErrorWithData(i18n.ErrInvalidChangeNotifyPoll, map[string]interface{}{"Error": err.Error()})
		}
		if d < MinChangeNotifyPoll {
			return i18n.NewI18nError(i18n.ErrChangeNotifyPollTooShort)
		}
	}

	return nil
}

// ValidateConnectionOptions validates the connection options input.
// It validates all nested option types.
func ValidateConnectionOptions(opts *ConnectionOptionsInput) error {
	if opts == nil {
		return nil
	}
	return ValidateCacheOptions(opts.Cache)
}

// ConnectionOptionsInputToModel converts ConnectionOptionsInput to ConnectionOptions for storage.
func ConnectionOptionsInputToModel(input *ConnectionOptionsInput) *ConnectionOptions {
	if input == nil {
		return nil
	}

	result := &ConnectionOptions{}

	if input.Cache != nil {
		result.Cache = &ConnectionCacheOptions{
			Enabled:          input.Cache.Enabled,
			InfoAge:          input.Cache.InfoAge,
			ChangeNotifyPoll: input.Cache.ChangeNotifyPoll,
		}
	}

	return result
}

// IsCacheEnabled returns whether cache is enabled for the given options.
func IsCacheEnabled(opts *ConnectionOptions) bool {
	return opts != nil && opts.Cache != nil && opts.Cache.Enabled
}
