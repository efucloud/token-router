import { getToken } from '@/utils/auth';

export type GatewayModel = {
  id: string;
  object: 'model';
  owned_by: string;
  display_name?: string;
  description?: string;
  modality?: string;
  context_window?: number;
  max_output_tokens?: number;
  capabilities?: string[];
};

export type ChatMessage = {
  role: 'system' | 'user' | 'assistant';
  content: string;
};

export type ChatToolCall = {
  id: string;
  type: 'function';
  function: {
    name: string;
    arguments: string;
  };
};

export type ChatRequestMessage =
  | ChatMessage
  | {
      role: 'assistant';
      content: string | null;
      tool_calls: ChatToolCall[];
    }
  | {
      role: 'tool';
      content: string;
      tool_call_id: string;
      name?: string;
    };

export type ChatToolDefinition = {
  type: 'function';
  function: {
    name: string;
    description?: string;
    parameters: Record<string, unknown>;
  };
};

type GatewayError = {
  error?: { message?: string; code?: string };
};

const retryableStatuses = new Set([
  408, 409, 425, 429, 500, 502, 503, 504, 529,
]);

const retryAfterMillis = (value: string | null) => {
  if (!value) return undefined;
  const seconds = Number(value);
  if (Number.isFinite(seconds)) return Math.max(0, seconds * 1000);
  const date = Date.parse(value);
  if (Number.isNaN(date)) return undefined;
  return Math.max(0, date - Date.now());
};

export class GatewayRequestError extends Error {
  status: number;
  retryAfter?: number;
  retryable: boolean;

  constructor(message: string, status: number, retryAfter?: number) {
    super(message);
    this.name = 'GatewayRequestError';
    this.status = status;
    this.retryAfter = retryAfter;
    this.retryable = retryableStatuses.has(status);
  }
}

const gatewayRequest = async <T>(
  path: string,
  init?: RequestInit,
): Promise<T> => {
  const token = getToken();
  if (!token) throw new Error('登录状态已失效，请重新登录');
  const response = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token.access_token}`,
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as GatewayError;
    throw new GatewayRequestError(
      body.error?.message || `网关请求失败（HTTP ${response.status}）`,
      response.status,
      retryAfterMillis(response.headers.get('Retry-After')),
    );
  }
  return response.json() as Promise<T>;
};

export const listGatewayModels = async () => {
  const result = await gatewayRequest<{ data: GatewayModel[] }>('/v1/models');
  return result.data;
};

type ChatCompletionOptions = {
  conversationID?: string;
  toolChoice?: 'auto' | 'none';
  onContent?: (content: string) => void;
  onActivity?: () => void;
  signal?: AbortSignal;
};

export const createChatCompletion = async (
  model: string,
  messages: ChatRequestMessage[],
  options: ChatCompletionOptions = {},
) => {
  const token = getToken();
  if (!token) throw new Error('登录状态已失效，请重新登录');
  const response = await fetch('/v1/chat/completions', {
    method: 'POST',
    headers: {
      Accept: 'application/x-ndjson',
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token.access_token}`,
      ...(options.conversationID
        ? { 'X-Token-Router-Conversation-Id': options.conversationID }
        : {}),
    },
    body: JSON.stringify({
      model,
      messages,
      stream: true,
      ...(options.toolChoice ? { tool_choice: options.toolChoice } : {}),
    }),
    signal: options.signal,
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as GatewayError;
    throw new GatewayRequestError(
      body.error?.message || `网关请求失败（HTTP ${response.status}）`,
      response.status,
      retryAfterMillis(response.headers.get('Retry-After')),
    );
  }
  if (!response.body) throw new Error('浏览器不支持流式响应');

  type ChatChunk = {
    error?: { message?: string };
    choices?: Array<{
      delta?: {
        content?: string;
        tool_calls?: Array<Partial<ChatToolCall> & { index?: number }>;
      };
      message?: {
        content?: string;
        tool_calls?: ChatToolCall[];
      };
    }>;
    usage?: {
      prompt_tokens?: number;
      completion_tokens?: number;
      total_tokens?: number;
    };
  };

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  const toolCalls = new Map<number, ChatToolCall>();
  let pending = '';
  let content = '';
  let totalTokens: number | undefined;
  const consumeLine = (line: string) => {
    const trimmed = line.trim();
    if (!trimmed) return;
    let chunk: ChatChunk;
    try {
      chunk = JSON.parse(trimmed) as ChatChunk;
    } catch {
      throw new GatewayRequestError('模型流返回了无效数据', 502);
    }
    if (chunk.error?.message)
      throw new GatewayRequestError(chunk.error.message, 502);
    const choice = chunk.choices?.[0];
    const delta = choice?.delta?.content || choice?.message?.content || '';
    if (delta) {
      options.onActivity?.();
      content += delta;
      options.onContent?.(content);
    }
    const incomingCalls =
      choice?.delta?.tool_calls || choice?.message?.tool_calls || [];
    for (const [position, incoming] of incomingCalls.entries()) {
      options.onActivity?.();
      const index = 'index' in incoming ? incoming.index || 0 : position;
      const current = toolCalls.get(index) || {
        id: '',
        type: 'function' as const,
        function: { name: '', arguments: '' },
      };
      toolCalls.set(index, {
        id: incoming.id || current.id,
        type: 'function',
        function: {
          name: incoming.function?.name || current.function.name,
          arguments:
            current.function.arguments + (incoming.function?.arguments || ''),
        },
      });
    }
    if (chunk.usage?.total_tokens !== undefined)
      totalTokens = chunk.usage.total_tokens;
  };

  while (true) {
    const { done, value } = await reader.read();
    pending += decoder.decode(value, { stream: !done });
    const lines = pending.split(/\r?\n/);
    pending = lines.pop() || '';
    for (const line of lines) consumeLine(line);
    if (done) break;
  }
  consumeLine(pending);
  return {
    content,
    totalTokens,
    toolCalls: [...toolCalls.entries()]
      .sort(([left], [right]) => left - right)
      .map(([, call]) => call),
  };
};
