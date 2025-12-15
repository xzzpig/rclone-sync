import * as m from '@/paraglide/messages.js';
import { getLocale } from '@/paraglide/runtime';
import axios, { AxiosError } from 'axios';

// Custom error class with structured error information
export class ApiError extends Error {
  public status?: number;
  public details?: string;

  constructor(message: string, status?: number, details?: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.details = details;
  }
}

type ResponseError = {
  response?: {
    data?: {
      message?: string;
      error?: string;
      details?: string;
    };
    status?: number;
  };
};

function isResponseError(err: unknown): err is ResponseError {
  return typeof err === 'object' && err !== null && 'response' in err;
}

// Extract error message from API response
export function extractErrorMessage(err: unknown): string {
  if (isResponseError(err) && err.response?.data?.message) {
    return err.response.data.message;
  } else if (isResponseError(err) && err.response?.data?.error) {
    return err.response.data.error;
  } else if (err instanceof Error) {
    return err.message;
  }
  return m.error_unknownError();
}

// Extract error details from API response
export function extractErrorDetails(err: unknown): string | undefined {
  if (isResponseError(err)) {
    return err.response?.data?.details;
  }
  return undefined;
}

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add Accept-Language header
api.interceptors.request.use(
  (config) => {
    // Add Accept-Language header based on current locale
    const locale = getLocale();
    config.headers['Accept-Language'] = locale;
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    const message = extractErrorMessage(error);
    const details = extractErrorDetails(error);
    const status = error.response?.status;

    // Create a structured error
    const apiError = new ApiError(message, status, details);

    return Promise.reject(apiError);
  }
);

export default api;
