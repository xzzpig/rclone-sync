import * as m from '@/paraglide/messages.js';
import { type HookEvent, type HookType, type HookOnError } from '@/lib/types';

export const getEventLabel = (event?: HookEvent | string | null) => {
  if (event === 'ON_START') return m.hook_event_onStart();
  if (event === 'ON_SUCCESS') return m.hook_event_onSuccess();
  if (event === 'ON_FAILURE') return m.hook_event_onFailure();
  if (event === 'ON_END') return m.hook_event_onEnd();
  return typeof event === 'string' ? event : m.common_select();
};

export const getTypeLabel = (type?: HookType | string | null) => {
  if (type === 'HTTP') return m.hook_type_http();
  if (type === 'COMMAND') return m.hook_type_command();
  return typeof type === 'string' ? type : m.common_select();
};

export const getOnErrorLabel = (onError?: HookOnError | string | null) => {
  if (onError === 'IGNORE') return m.hook_onError_ignore();
  if (onError === 'CANCEL') return m.hook_onError_cancel();
  if (onError === 'FATAL') return m.hook_onError_fatal();
  return typeof onError === 'string' ? onError : m.common_select();
};
