// SSE Client Utility

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type SSESubscriptionHandler<T = any> = (data: T) => void;

interface SSEConnection {
  close: () => void;
  // Generic handler for all events
  onMessage: (handler: SSESubscriptionHandler) => void;
  // Specific event handler (e.g., 'job_update')
  on: (event: string, handler: SSESubscriptionHandler) => void;
  // Remove event handler
  off: (event: string, handler: SSESubscriptionHandler) => void;
}

export class SSEClient {
  private url: string;
  private eventSource: EventSource | null = null;
  private listeners: Map<string, Set<SSESubscriptionHandler>> = new Map();
  // Track native event listeners to properly remove them
  private nativeListeners: Map<string, (e: MessageEvent) => void> = new Map();
  private reconnectInterval: number = 3000;
  private shouldReconnect: boolean = true;
  private reconnectTimeoutId: number | null = null;

  constructor(url: string = '/api/events') {
    this.url = url;
  }

  public connect(): SSEConnection {
    this.shouldReconnect = true;
    this.setupEventSource();

    return {
      close: () => this.disconnect(),
      onMessage: (handler) => this.addEventListener('message', handler),
      on: (event, handler) => this.addEventListener(event, handler),
      off: (event, handler) => this.removeEventListener(event, handler),
    };
  }

  public disconnect() {
    this.shouldReconnect = false;

    // Clear any pending reconnect timeout
    if (this.reconnectTimeoutId !== null) {
      clearTimeout(this.reconnectTimeoutId);
      this.reconnectTimeoutId = null;
    }

    if (this.eventSource) {
      // Remove all native listeners before closing
      this.nativeListeners.forEach((listener, event) => {
        this.eventSource?.removeEventListener(event, listener);
      });
      this.nativeListeners.clear();

      this.eventSource.close();
      this.eventSource = null;
    }
  }

  private setupEventSource() {
    // Close existing connection if any
    if (this.eventSource) {
      this.nativeListeners.forEach((listener, event) => {
        this.eventSource?.removeEventListener(event, listener);
      });
      this.nativeListeners.clear();
      this.eventSource.close();
    }

    this.eventSource = new EventSource(this.url);

    this.eventSource.onopen = () => {
      console.info(`SSE Connected to ${this.url}`);
    };

    this.eventSource.onerror = (err) => {
      console.error('SSE Error:', err);
      // If connection is closed and we should reconnect
      if (this.eventSource?.readyState === EventSource.CLOSED && this.shouldReconnect) {
        console.info(`SSE reconnecting in ${this.reconnectInterval}ms...`);
        this.reconnectTimeoutId = window.setTimeout(() => {
          this.setupEventSource();
        }, this.reconnectInterval);
      }
    };

    // Re-register all event listeners from this.listeners
    this.listeners.forEach((handlers, event) => {
      if (handlers.size > 0) {
        this.registerNativeListener(event);
      }
    });
  }

  private registerNativeListener(event: string) {
    // Don't register the same event twice
    if (this.nativeListeners.has(event) || !this.eventSource) {
      return;
    }

    const listener = (e: MessageEvent) => {
      this.dispatchEvent(event, e.data);
    };

    this.nativeListeners.set(event, listener);

    if (event === 'message') {
      this.eventSource.onmessage = listener;
    } else {
      this.eventSource.addEventListener(event, listener);
    }
  }

  private unregisterNativeListener(event: string) {
    const listener = this.nativeListeners.get(event);
    if (!listener || !this.eventSource) {
      return;
    }

    if (event === 'message') {
      this.eventSource.onmessage = null;
    } else {
      this.eventSource.removeEventListener(event, listener);
    }

    this.nativeListeners.delete(event);
  }

  private addEventListener(event: string, handler: SSESubscriptionHandler) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }

    const handlers = this.listeners.get(event)!;
    const wasEmpty = handlers.size === 0;
    handlers.add(handler);

    // Register native listener if this is the first handler for this event
    if (wasEmpty && this.eventSource) {
      this.registerNativeListener(event);
    }
  }

  private removeEventListener(event: string, handler: SSESubscriptionHandler) {
    const handlers = this.listeners.get(event);
    if (!handlers) {
      return;
    }

    handlers.delete(handler);

    // Unregister native listener if no more handlers for this event
    if (handlers.size === 0) {
      this.unregisterNativeListener(event);
      this.listeners.delete(event);
    }
  }

  private dispatchEvent(event: string, rawData: string) {
    const handlers = this.listeners.get(event);
    if (!handlers || handlers.size === 0) {
      return;
    }

    let data = rawData;
    try {
      data = JSON.parse(rawData);
    } catch {
      // Keep as string if not JSON
    }

    handlers.forEach((handler) => {
      try {
        handler(data);
      } catch (error) {
        console.error(`Error in SSE handler for event '${event}':`, error);
      }
    });
  }
}

export const sseClient = new SSEClient();
