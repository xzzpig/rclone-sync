/**
 * Date formatting utilities with locale support
 * Provides localized date, time, and relative time formatting
 */

import { getLocale } from '@/paraglide/runtime';
import * as m from '@/paraglide/messages';

export interface RelativeTimeFormatOptions {
  numeric?: 'always' | 'auto';
  style?: 'long' | 'short' | 'narrow';
}

/**
 * Format a date according to the current locale
 * @param date - The date to format
 * @param options - Intl.DateTimeFormat options
 * @returns Formatted date string
 */
export function formatDate(
  date: Date | string | number,
  options?: Intl.DateTimeFormatOptions
): string {
  const dateObj = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date;

  const locale = getLocale();

  const defaultOptions: Intl.DateTimeFormatOptions = {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    ...options,
  };

  return new Intl.DateTimeFormat(locale, defaultOptions).format(dateObj);
}

/**
 * Format a time according to the current locale
 * @param date - The date/time to format
 * @param options - Intl.DateTimeFormat options
 * @returns Formatted time string
 */
export function formatTime(
  date: Date | string | number,
  options?: Intl.DateTimeFormatOptions
): string {
  const dateObj = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date;

  const locale = getLocale();

  const defaultOptions: Intl.DateTimeFormatOptions = {
    hour: '2-digit',
    minute: '2-digit',
    ...options,
  };

  return new Intl.DateTimeFormat(locale, defaultOptions).format(dateObj);
}

/**
 * Format a date and time according to the current locale
 * @param date - The date/time to format
 * @param options - Intl.DateTimeFormat options
 * @returns Formatted date and time string
 */
export function formatDateTime(
  date: Date | string | number,
  options?: Intl.DateTimeFormatOptions
): string {
  const dateObj = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date;

  const locale = getLocale();

  const defaultOptions: Intl.DateTimeFormatOptions = {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    ...options,
  };

  return new Intl.DateTimeFormat(locale, defaultOptions).format(dateObj);
}

/**
 * Format a relative time (e.g., "2 minutes ago", "in 3 hours")
 * Uses custom translations for common relative time units
 * @param date - The date to compare to now
 * @returns Localized relative time string
 */
export function formatRelativeTime(date: Date | string | number): string {
  const dateObj = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date;

  const now = new Date();
  const diffMs = dateObj.getTime() - now.getTime();
  const diffSec = Math.round(diffMs / 1000);
  const diffMin = Math.round(diffSec / 60);
  const diffHour = Math.round(diffMin / 60);
  const diffDay = Math.round(diffHour / 24);
  const diffWeek = Math.round(diffDay / 7);
  const diffMonth = Math.round(diffDay / 30);
  const diffYear = Math.round(diffDay / 365);

  // Use custom translations for common cases
  if (Math.abs(diffSec) < 10) {
    return m.time_justNow();
  }

  if (Math.abs(diffSec) < 60) {
    return diffSec > 0
      ? m.time_secondsFromNow({ seconds: diffSec.toString() })
      : m.time_secondsAgo({ seconds: Math.abs(diffSec).toString() });
  }

  if (Math.abs(diffMin) < 60) {
    return diffMin > 0
      ? m.time_minutesFromNow({ minutes: diffMin.toString() })
      : m.time_minutesAgo({ minutes: Math.abs(diffMin).toString() });
  }

  if (Math.abs(diffHour) < 24) {
    return diffHour > 0
      ? m.time_hoursFromNow({ hours: diffHour.toString() })
      : m.time_hoursAgo({ hours: Math.abs(diffHour).toString() });
  }

  if (Math.abs(diffDay) < 7) {
    return diffDay > 0
      ? m.time_daysFromNow({ days: diffDay.toString() })
      : m.time_daysAgo({ days: Math.abs(diffDay).toString() });
  }

  if (Math.abs(diffWeek) < 5) {
    return diffWeek > 0
      ? m.time_weeksFromNow({ weeks: diffWeek.toString() })
      : m.time_weeksAgo({ weeks: Math.abs(diffWeek).toString() });
  }

  if (Math.abs(diffMonth) < 12) {
    return diffMonth > 0
      ? m.time_monthsFromNow({ months: diffMonth.toString() })
      : m.time_monthsAgo({ months: Math.abs(diffMonth).toString() });
  }

  return diffYear > 0
    ? m.time_yearsFromNow({ years: diffYear.toString() })
    : m.time_yearsAgo({ years: Math.abs(diffYear).toString() });
}

/**
 * Format a duration in milliseconds to a human-readable string
 * @param durationMs - Duration in milliseconds
 * @returns Formatted duration string
 */
export function formatDuration(durationMs: number): string {
  const seconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    return m.time_duration_days({ days: days.toString() });
  }
  if (hours > 0) {
    return m.time_duration_hours({ hours: hours.toString() });
  }
  if (minutes > 0) {
    return m.time_duration_minutes({ minutes: minutes.toString() });
  }
  return m.time_duration_seconds({ seconds: seconds.toString() });
}
