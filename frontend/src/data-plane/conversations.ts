import { getToken } from '@/utils/auth';

export type ConversationSummary = {
  id: string;
  title: string;
  model: string;
  createdAt: string;
  updatedAt: string;
};

export type StoredConversationMessage = {
  id: string;
  conversationId: string;
  role: 'system' | 'user' | 'assistant';
  content: string;
  totalTokens: number;
  sequence: number;
  createdAt: string;
};

export type ConversationDetail = ConversationSummary & {
  messages: StoredConversationMessage[];
};

type ControlPlaneError = {
  alert?: string;
  detail?: string;
  message?: string;
};

const conversationRequest = async <T>(
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
    const body = (await response
      .json()
      .catch(() => ({}))) as ControlPlaneError;
    throw new Error(
      body.alert ||
        body.detail ||
        body.message ||
        `请求失败（HTTP ${response.status}）`,
    );
  }
  return response.json() as Promise<T>;
};

const base = '/api/v1/chat/conversations';

export const listConversations = () =>
  conversationRequest<ConversationSummary[]>(base);

export const createConversation = (model: string) =>
  conversationRequest<ConversationSummary>(base, {
    method: 'POST',
    body: JSON.stringify({ model }),
  });

export const getConversation = (id: string) =>
  conversationRequest<ConversationDetail>(`${base}/${id}`);

export const updateConversation = (
  id: string,
  input: { model: string; title?: string },
) =>
  conversationRequest<ConversationSummary>(`${base}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });

export const appendConversationMessage = (
  id: string,
  input: {
    role: 'system' | 'user' | 'assistant';
    content: string;
    totalTokens?: number;
  },
) =>
  conversationRequest<StoredConversationMessage>(`${base}/${id}/messages`, {
    method: 'POST',
    body: JSON.stringify(input),
  });

export const deleteConversation = (id: string) =>
  conversationRequest<string>(`${base}/${id}`, { method: 'DELETE' });
