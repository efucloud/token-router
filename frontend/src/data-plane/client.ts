import { getToken } from '@/utils/auth';

export type GatewayModel = {
  id: string;
  object: 'model';
  owned_by: string;
};

export type ChatMessage = {
  role: 'system' | 'user' | 'assistant';
  content: string;
};

type GatewayError = {
  error?: { message?: string; code?: string };
};

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
    throw new Error(
      body.error?.message || `网关请求失败（HTTP ${response.status}）`,
    );
  }
  return response.json() as Promise<T>;
};

export const listGatewayModels = async () => {
  const result = await gatewayRequest<{ data: GatewayModel[] }>(
    '/v1/models',
  );
  return result.data;
};

export const createChatCompletion = async (
  model: string,
  messages: ChatMessage[],
  onContent?: (content: string) => void,
) => {
  const token = getToken();
  if (!token) throw new Error('登录状态已失效，请重新登录');
  const response = await fetch('/v1/chat/completions', {
    method: 'POST',
    headers: {
      Accept: 'application/x-ndjson',
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token.access_token}`,
    },
    body: JSON.stringify({ model, messages, stream: true }),
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as GatewayError;
    throw new Error(
      body.error?.message || `网关请求失败（HTTP ${response.status}）`,
    );
  }
  if (!response.body) throw new Error('浏览器不支持流式响应');

  type ChatChunk = {
    error?: { message?: string };
    choices?: Array<{
      delta?: { content?: string };
      message?: ChatMessage;
    }>;
    usage?: {
      prompt_tokens?: number;
      completion_tokens?: number;
      total_tokens?: number;
    };
  };

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let pending = '';
  let content = '';
  let totalTokens: number | undefined;
  const consumeLine = (line: string) => {
    const trimmed = line.trim();
    if (!trimmed) return;
    const chunk = JSON.parse(trimmed) as ChatChunk;
    if (chunk.error?.message) throw new Error(chunk.error.message);
    const delta =
      chunk.choices?.[0]?.delta?.content ||
      chunk.choices?.[0]?.message?.content ||
      '';
    if (delta) {
      content += delta;
      onContent?.(content);
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
  };
};
