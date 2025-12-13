import api from '@/lib/api';
import type { FileEntry } from '@/lib/types';

export const listLocalFiles = async (path: string, blacklist?: string[]) => {
  const params = new URLSearchParams({ path });
  if (blacklist && blacklist.length > 0) {
    params.append('blacklist', blacklist.join(','));
  }
  const response = await api.get<FileEntry[]>(`/files/local?${params}`);
  return response.data;
};

export const listRemoteFiles = async (remoteName: string, path: string) => {
  const params = new URLSearchParams({ path });
  const response = await api.get<FileEntry[]>(`/files/remote/${remoteName}?${params}`);
  return response.data;
};
