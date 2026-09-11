import { getToken } from '@/utils/auth';

export type ConversationSummary = {
  id: string;
  title: string;
  model: string;
  skills: string[];
  mcpServers: string[];
  createdAt: string;
  updatedAt: string;
};

export type ChatRuntimePolicy = {
  contextWindowTokens: number;
  compactThreshold: number;
  compactKeepRecent: number;
  maxRetries: number;
  retryBaseMillis: number;
  retryMaxMillis: number;
  maxToolRounds: number;
};

export type ChatSkillCapability = {
  name: string;
  description: string;
  content: string;
};

export type ChatMCPToolCapability = {
  name: string;
  description: string;
  inputSchema: Record<string, unknown>;
};

export type ChatMCPServerCapability = {
  name: string;
  kind: 'builtin' | 'mcp';
  status: 'connected' | 'failed';
  defaultEnabled: boolean;
  error?: string;
  tools: ChatMCPToolCapability[];
};

export type ChatCapabilities = {
  policy: ChatRuntimePolicy;
  skills: ChatSkillCapability[];
  mcpServers: ChatMCPServerCapability[];
  issues: string[];
};

export type ChatWorkspaceEntry = {
  name: string;
  path: string;
  type: 'file' | 'directory' | 'symlink';
  size: number;
  updatedAt: string;
};

export type ChatWorkspaceListing = {
  path: string;
  entries: ChatWorkspaceEntry[];
  maxUploadBytes: number;
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
      Authorization: `Bearer ${token.access_token}`,
      ...(init?.body instanceof FormData
        ? {}
        : { 'Content-Type': 'application/json' }),
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ControlPlaneError;
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

export const createConversation = (
  model: string,
  skills: string[] = [],
  mcpServers: string[] = [],
) =>
  conversationRequest<ConversationSummary>(base, {
    method: 'POST',
    body: JSON.stringify({ model, skills, mcpServers }),
  });

export const getConversation = (id: string) =>
  conversationRequest<ConversationDetail>(`${base}/${id}`);

export const updateConversation = (
  id: string,
  input: {
    model: string;
    title?: string;
    skills: string[];
    mcpServers: string[];
  },
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

export const getChatCapabilities = async () => {
  const result = await conversationRequest<ChatCapabilities>(
    '/api/v1/chat/capabilities',
  );
  return {
    ...result,
    mcpServers: (result.mcpServers || []).map((server) => ({
      ...server,
      kind: server.kind || 'mcp',
      defaultEnabled: server.defaultEnabled || false,
      tools: server.tools || [],
    })),
  };
};

export const callChatMCPTool = (
  server: string,
  tool: string,
  args: Record<string, unknown>,
) =>
  conversationRequest<{ content: string; isError: boolean }>(
    `/api/v1/chat/mcp/${encodeURIComponent(server)}/tools/${encodeURIComponent(tool)}/call`,
    {
      method: 'POST',
      body: JSON.stringify({ arguments: args }),
    },
  );

const workspaceBase = '/api/v1/chat/workspace';

export const listChatWorkspace = (path = '') =>
  conversationRequest<ChatWorkspaceListing>(
    `${workspaceBase}?path=${encodeURIComponent(path)}`,
  );

export const uploadChatWorkspaceFile = (path: string, file: File) => {
  const body = new FormData();
  body.append('file', file);
  return conversationRequest<ChatWorkspaceEntry>(
    `${workspaceBase}/upload?path=${encodeURIComponent(path)}`,
    { method: 'POST', body },
  );
};

export const downloadChatWorkspaceFile = async (path: string) => {
  const token = getToken();
  if (!token) throw new Error('登录状态已失效，请重新登录');
  const response = await fetch(
    `${workspaceBase}/download?path=${encodeURIComponent(path)}`,
    { headers: { Authorization: `Bearer ${token.access_token}` } },
  );
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ControlPlaneError;
    throw new Error(
      body.alert ||
        body.detail ||
        body.message ||
        `请求失败（HTTP ${response.status}）`,
    );
  }
  return response.blob();
};
