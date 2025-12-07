import { API_BASE } from './config';

export interface Provider {
    name: string;
    description: string;
    options?: any[]; // Simplified for now
}

export interface RemoteInfo {
    [key: string]: string;
}

export const fetchRemotes = async (): Promise<string[]> => {
    const res = await fetch(`${API_BASE}/remotes`);
    if (!res.ok) throw new Error('Failed to fetch remotes');
    return res.json();
};

export const fetchRemoteInfo = async (name: string): Promise<RemoteInfo> => {
    const res = await fetch(`${API_BASE}/remotes/${name}`);
    if (!res.ok) throw new Error('Failed to fetch remote info');
    return res.json();
};

export const createRemote = async (name: string, config: RemoteInfo): Promise<void> => {
    const res = await fetch(`${API_BASE}/remotes/${name}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
    });
    if (!res.ok) throw new Error('Failed to create remote');
};

export const deleteRemote = async (name: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/remotes/${name}`, {
        method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete remote');
};

export const fetchProviders = async (): Promise<Provider[]> => {
    const res = await fetch(`${API_BASE}/providers`);
    if (!res.ok) throw new Error('Failed to fetch providers');
    return res.json();
};

export const fetchProviderOptions = async (name: string): Promise<Provider> => {
    const res = await fetch(`${API_BASE}/providers/${name}`);
    if (!res.ok) throw new Error('Failed to fetch provider options');
    return res.json();
};
