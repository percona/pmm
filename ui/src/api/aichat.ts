export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system' | 'tool' | 'tool_approval';
  content: string;
  timestamp: string;
  tool_calls?: ToolCall[];
  tool_executions?: ToolExecution[];
  attachments?: FileAttachment[];
  approval_request?: {
    request_id: string;
    tool_calls: ToolCall[];
    processed?: boolean; // To hide buttons after approval/denial
  };
}

export interface ToolCall {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

export interface ToolExecution {
  id: string;
  tool_name: string;
  arguments: string;
  result: string;
  error?: string;
  start_time: string;
  end_time: string;
  duration_ms: number;
}

export interface ChatRequest {
  message: string;
  session_id?: string;
  context?: Record<string, string>;
}

export interface FileAttachment {
  id?: string;
  filename: string;
  content: string; // base64 encoded
  mime_type: string;
  size: number;
  path?: string;
}

export interface ChatRequestWithFiles {
  message: string;
  session_id?: string;
  attachments?: FileAttachment[];
}

export interface ChatResponse {
  message?: ChatMessage;
  session_id: string;
  error?: string;
}

export interface ChatHistory {
  session_id: string;
  messages: ChatMessage[];
}

export interface ChatSession {
  id: string;
  user_id: string;
  title: string;
  created_at: string;
  updated_at: string;
}

export interface SessionListResponse {
  sessions: ChatSession[];
  pagination: {
    page: number;
    limit: number;
    offset: number;
  };
}

export interface CreateSessionRequest {
  title: string;
}

export interface UpdateSessionRequest {
  title: string;
}

export interface MCPTool {
  name: string;
  description: string;
  input_schema: Record<string, any>;
  server: string;
}

export interface MCPToolsResponse {
  tools: MCPTool[];
  force_refresh?: boolean;
}

export interface ToolApprovalRequest {
  session_id: string;
  tool_calls: ToolCall[];
  request_id: string;
}

export interface ToolApprovalResponse {
  session_id: string;
  request_id: string;
  approved: boolean;
  approved_ids?: string[]; // For selective approval
}

export interface StreamMessage {
  type: 'message' | 'tool_call' | 'tool_execution' | 'tool_approval_request' | 'error' | 'done';
  content?: string;
  session_id: string;
  error?: string;
  tool_calls?: ToolCall[];
  tool_executions?: ToolExecution[];
  request_id?: string; // For tool approval requests
}

class AIChatAPI {
  private baseURL: string;

  constructor() {
    // Use relative URL - nginx will proxy /v1/chat/* to aichat-backend
    this.baseURL = '/v1/chat';
  }

  // Get default headers for API requests
  private getDefaultHeaders(): Record<string, string> {
    return {
      'Content-Type': 'application/json',
    };
  }

  /**
   * Converts a base64 string to a Blob object
   * @param base64 - The base64 encoded string
   * @param mimeType - The MIME type for the blob
   * @returns A Blob object
   */
  private base64ToBlob(base64: string, mimeType: string): Blob {
    const byteCharacters = atob(base64);
    const byteNumbers = new Array(byteCharacters.length);
    for (let i = 0; i < byteCharacters.length; i++) {
      byteNumbers[i] = byteCharacters.charCodeAt(i);
    }
    const byteArray = new Uint8Array(byteNumbers);
    return new Blob([byteArray], { type: mimeType });
  }

  /**
   * Conditionally logs debug messages only in development environment
   * @param message - The message to log
   * @param optionalParams - Additional parameters to log
   */
  private debugLog(message?: any, ...optionalParams: any[]): void {
    // Check for development environment in a browser-safe way
    const isDevelopment = import.meta.env?.DEV === true ||
                         window.location.hostname === 'localhost' ||
                         window.location.hostname === '127.0.0.1' ||
                         window.location.port === '3000' ||
                         window.location.port === '5173';
    
    if (isDevelopment) {
      console.log(message, ...optionalParams);
    }
  }

  async sendMessage(request: ChatRequest): Promise<ChatResponse> {
    const url = `${this.baseURL}/send`;
    const response = await fetch(url, {
      method: 'POST',
      headers: this.getDefaultHeaders(),
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`Failed to send message: HTTP ${response.status} ${response.statusText} - POST ${url} - Session: ${request.session_id || 'new'}`);
    }

    return response.json();
  }

  async sendMessageWithFiles(request: ChatRequestWithFiles): Promise<ChatResponse> {
    const formData = new FormData();
    
    // Add message and session_id as form fields
    formData.append('message', request.message);
    if (request.session_id) {
      formData.append('session_id', request.session_id);
    }

    // Add files as form data
    if (request.attachments) {
      request.attachments.forEach((attachment, index) => {
        // Convert base64 to blob using helper function
        const blob = this.base64ToBlob(attachment.content, attachment.mime_type);
        
        // Add file to form data with consistent field naming pattern
        formData.append(`file${index}`, blob, attachment.filename);
      });
    }

    const url = `${this.baseURL}/send-with-files`;
    const response = await fetch(url, {
      method: 'POST',
      // Don't set Content-Type header - let browser set it with boundary for multipart
      body: formData,
    });

    if (!response.ok) {
      const fileCount = request.attachments?.length || 0;
      const fileNames = request.attachments?.map(a => a.filename).join(', ') || 'none';
      throw new Error(`Failed to send message with files: HTTP ${response.status} ${response.statusText} - POST ${url} - Session: ${request.session_id || 'new'} - Files: ${fileCount} (${fileNames})`);
    }

    return response.json();
  }

  async getHistory(sessionId: string): Promise<ChatHistory> {
    const url = `${this.baseURL}/sessions/${encodeURIComponent(sessionId)}/messages`;
    const response = await fetch(url, {
      headers: this.getDefaultHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to get chat history: HTTP ${response.status} ${response.statusText} - GET ${url} - Session: ${sessionId}`);
    }

    const data = await response.json();
    this.debugLog('🔍 API: getHistory response:', data);
    
    // Debug attachments specifically
    if (data.messages) {
      const messagesWithAttachments = data.messages.filter((msg: ChatMessage) => msg.attachments && msg.attachments.length > 0);
      this.debugLog('🔍 API: Messages with attachments:', messagesWithAttachments);
    }

    return {
      session_id: sessionId,
      messages: data.messages || [],
    };
  }

  async clearHistory(sessionId: string): Promise<void> {
    const url = `${this.baseURL}/sessions/${encodeURIComponent(sessionId)}/messages`;
    const response = await fetch(url, {
      method: 'DELETE',
      headers: this.getDefaultHeaders(),
    });

    if (!response.ok) {
      throw new Error(`Failed to clear chat history: HTTP ${response.status} ${response.statusText} - DELETE ${url} - Session: ${sessionId}`);
    }
  }

  async getMCPTools(forceRefresh?: boolean): Promise<MCPToolsResponse> {
    const url = new URL(`${this.baseURL}/mcp/tools`, window.location.origin);
    if (forceRefresh) {
      url.searchParams.set('force_refresh', 'true');
    }
    
    this.debugLog(`🌐 API: Making request to ${url.toString()}`);
    
    const response = await fetch(url.toString());
    
    this.debugLog(`🌐 API: Response status: ${response.status} ${response.statusText}`);
    
    if (!response.ok) {
      const errorText = await response.text();
      console.error(`🌐 API: Error response body:`, errorText);
      throw new Error(`Failed to get MCP tools: HTTP ${response.status} ${response.statusText} - GET ${url.toString()} - Force refresh: ${forceRefresh || false} - Response: ${errorText}`);
    }

    const data = await response.json();
    this.debugLog(`🌐 API: Response data:`, data);
    return data;
  }

  // New streaming pattern: initiate stream and connect separately
  async initiateStream(sessionId: string, message: string): Promise<{
    stream_id: string;
    session_id: string;
    stream_url: string;
  }> {
    const url = `${this.baseURL}/send?streaming=true`;
    const requestBody = {
      message,
      session_id: sessionId,
    };
    const response = await fetch(url, {
      method: 'POST',
      headers: this.getDefaultHeaders(),
      body: JSON.stringify(requestBody),
    });

    if (!response.ok) {
      throw new Error(`Failed to initiate stream: HTTP ${response.status} ${response.statusText} - POST ${url} - Session: ${sessionId} - Message length: ${message.length}`);
    }

    return response.json();
  }

  // Connect to an existing stream by ID
  connectToStream(
    streamId: string,
    onMessage: (message: StreamMessage) => void,
    onError: (error: string) => void,
    onComplete: () => void
  ): () => void {
    const url = new URL(`${this.baseURL}/stream/${streamId}`, window.location.origin);
    const eventSource = new EventSource(url.toString());
    
    eventSource.onmessage = (event) => {
      try {
        const streamMessage: StreamMessage = JSON.parse(event.data);
        onMessage(streamMessage);
        
        if (streamMessage.type === 'done' || streamMessage.type === 'error') {
          eventSource.close();
          onComplete();
        }
      } catch (error) {
        console.error('Failed to parse stream message:', error);
        onError('Failed to parse response');
        eventSource.close();
        onComplete();
      }
    };

    eventSource.onerror = (event) => {
      console.error('EventSource failed:', event);
      onError('Connection failed');
      eventSource.close();
      onComplete();
    };

    // Return cleanup function
    return () => {
      eventSource.close();
    };
  }

  // High-level method that combines initiate + connect
  async streamChatWithSeparateEndpoints(
    sessionId: string,
    message: string,
    onMessage: (message: StreamMessage) => void,
    onError: (error: string) => void,
    onComplete: () => void
  ): Promise<() => void> {
    try {
      // Step 1: Initiate the stream
      const streamInfo = await this.initiateStream(sessionId, message);
      
      // Step 2: Connect to the stream
      return this.connectToStream(streamInfo.stream_id, onMessage, onError, onComplete);
    } catch (error) {
      onError(`Failed to initiate stream: ${error}`);
      onComplete();
      return () => {}; // Return empty cleanup function
    }
  }

  // Session management methods
  async listSessions(page: number = 1, limit: number = 20): Promise<SessionListResponse> {
    const url = `${this.baseURL}/sessions?page=${page}&limit=${limit}`;
    const response = await fetch(url, {
      headers: this.getDefaultHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to list sessions: HTTP ${response.status} ${response.statusText} - GET ${url} - Page: ${page}, Limit: ${limit}`);
    }

    return response.json();
  }

  async createSession(request: CreateSessionRequest): Promise<ChatSession> {
    const url = `${this.baseURL}/sessions`;
    const response = await fetch(url, {
      method: 'POST',
      headers: this.getDefaultHeaders(),
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`Failed to create session: HTTP ${response.status} ${response.statusText} - POST ${url} - Title: "${request.title}"`);
    }

    return response.json();
  }

  async getSession(sessionId: string): Promise<ChatSession> {
    const url = `${this.baseURL}/sessions/${encodeURIComponent(sessionId)}`;
    const response = await fetch(url, {
      headers: this.getDefaultHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to get session: HTTP ${response.status} ${response.statusText} - GET ${url} - Session: ${sessionId}`);
    }

    return response.json();
  }

  async updateSession(sessionId: string, request: UpdateSessionRequest): Promise<void> {
    const url = `${this.baseURL}/sessions/${encodeURIComponent(sessionId)}`;
    const response = await fetch(url, {
      method: 'PUT',
      headers: this.getDefaultHeaders(),
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`Failed to update session: HTTP ${response.status} ${response.statusText} - PUT ${url} - Session: ${sessionId} - Title: "${request.title}"`);
    }
  }

  async deleteSession(sessionId: string): Promise<void> {
    const url = `${this.baseURL}/sessions/${encodeURIComponent(sessionId)}`;
    const response = await fetch(url, {
      method: 'DELETE',
      headers: this.getDefaultHeaders(),
    });

    if (!response.ok) {
      throw new Error(`Failed to delete session: HTTP ${response.status} ${response.statusText} - DELETE ${url} - Session: ${sessionId}`);
    }
  }

  async getSessionMessages(sessionId: string, page: number = 1, limit: number = 50): Promise<{ messages: ChatMessage[] }> {
    const url = `${this.baseURL}/sessions/${encodeURIComponent(sessionId)}/messages?page=${page}&limit=${limit}`;
    const response = await fetch(url, {
      headers: this.getDefaultHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to get session messages: HTTP ${response.status} ${response.statusText} - GET ${url} - Session: ${sessionId} - Page: ${page}, Limit: ${limit}`);
    }

    return response.json();
  }
}

export const aiChatAPI = new AIChatAPI();