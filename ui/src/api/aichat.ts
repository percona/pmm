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

  async sendMessage(request: ChatRequest): Promise<ChatResponse> {
    const response = await fetch(`${this.baseURL}/send`, {
      method: 'POST',
      headers: this.getDefaultHeaders(),
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
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
        // Convert base64 back to blob for multipart upload
        const byteCharacters = atob(attachment.content);
        const byteNumbers = new Array(byteCharacters.length);
        for (let i = 0; i < byteCharacters.length; i++) {
          byteNumbers[i] = byteCharacters.charCodeAt(i);
        }
        const byteArray = new Uint8Array(byteNumbers);
        const blob = new Blob([byteArray], { type: attachment.mime_type });
        
        // Add file to form data with field name starting with "file"
        formData.append(`file${index}`, blob, attachment.filename);
      });
    }

    const response = await fetch(`${this.baseURL}/send-with-files`, {
      method: 'POST',
      // Don't set Content-Type header - let browser set it with boundary for multipart
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  async getHistory(sessionId: string): Promise<ChatHistory> {
    const response = await fetch(`${this.baseURL}/sessions/${encodeURIComponent(sessionId)}/messages`, {
      headers: this.getDefaultHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data = await response.json();
    console.log('🔍 API: getHistory response:', data);
    
    // Debug attachments specifically
    if (data.messages) {
      const messagesWithAttachments = data.messages.filter((msg: ChatMessage) => msg.attachments && msg.attachments.length > 0);
      console.log('🔍 API: Messages with attachments:', messagesWithAttachments);
    }

    return {
      session_id: sessionId,
      messages: data.messages || [],
    };
  }

  async clearHistory(sessionId: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/sessions/${encodeURIComponent(sessionId)}/messages`, {
      method: 'DELETE',
      headers: this.getDefaultHeaders(),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
  }

  async getMCPTools(forceRefresh?: boolean): Promise<MCPToolsResponse> {
    const url = new URL(`${this.baseURL}/mcp/tools`, window.location.origin);
    if (forceRefresh) {
      url.searchParams.set('force_refresh', 'true');
    }
    
    console.log(`🌐 API: Making request to ${url.toString()}`);
    
    const response = await fetch(url.toString());
    
    console.log(`🌐 API: Response status: ${response.status} ${response.statusText}`);
    
    if (!response.ok) {
      const errorText = await response.text();
      console.error(`🌐 API: Error response body:`, errorText);
      throw new Error(`HTTP error! status: ${response.status}, body: ${errorText}`);
    }

    const data = await response.json();
    console.log(`🌐 API: Response data:`, data);
    return data;
  }

  // Create a streaming connection for real-time chat
  createStreamConnection(sessionId: string, message: string): EventSource {
    const url = new URL(`${this.baseURL}/stream`, window.location.origin);
    url.searchParams.set('session_id', sessionId);
    url.searchParams.set('message', message);
    
    return new EventSource(url.toString());
  }

  // Stream chat with callback
  streamChat(
    sessionId: string,
    message: string,
    onMessage: (message: StreamMessage) => void,
    onError: (error: string) => void,
    onComplete: () => void
  ): () => void {
    const eventSource = this.createStreamConnection(sessionId, message);
    
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

  // Session management methods
  async listSessions(page: number = 1, limit: number = 20): Promise<SessionListResponse> {
    const response = await fetch(`${this.baseURL}/sessions?page=${page}&limit=${limit}`, {
      headers: this.getDefaultHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  async createSession(request: CreateSessionRequest): Promise<ChatSession> {
    const response = await fetch(`${this.baseURL}/sessions`, {
      method: 'POST',
      headers: this.getDefaultHeaders(),
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  async getSession(sessionId: string): Promise<ChatSession> {
    const response = await fetch(`${this.baseURL}/sessions/${encodeURIComponent(sessionId)}`, {
      headers: this.getDefaultHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  async updateSession(sessionId: string, request: UpdateSessionRequest): Promise<void> {
    const response = await fetch(`${this.baseURL}/sessions/${encodeURIComponent(sessionId)}`, {
      method: 'PUT',
      headers: this.getDefaultHeaders(),
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
  }

  async deleteSession(sessionId: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/sessions/${encodeURIComponent(sessionId)}`, {
      method: 'DELETE',
      headers: this.getDefaultHeaders(),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
  }

  async getSessionMessages(sessionId: string, page: number = 1, limit: number = 50): Promise<{ messages: ChatMessage[] }> {
    const response = await fetch(`${this.baseURL}/sessions/${encodeURIComponent(sessionId)}/messages?page=${page}&limit=${limit}`, {
      headers: this.getDefaultHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
  }
}

export const aiChatAPI = new AIChatAPI();