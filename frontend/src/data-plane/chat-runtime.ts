import type {
  ChatCapabilities,
  ChatMCPServerCapability,
  ChatRuntimePolicy,
  ChatSkillCapability,
} from './conversations';
import {
  GatewayRequestError,
  type ChatMessage,
  type ChatRequestMessage,
  type ChatToolDefinition,
} from './client';

export const SUMMARY_MARKER = '[token-router:summary]';

export const defaultChatPolicy: ChatRuntimePolicy = {
  contextWindowTokens: 32768,
  compactThreshold: 0.75,
  compactKeepRecent: 8,
  maxRetries: 3,
  retryBaseMillis: 800,
  retryMaxMillis: 8000,
  maxToolRounds: 8,
};

export const emptyChatCapabilities: ChatCapabilities = {
  policy: defaultChatPolicy,
  skills: [],
  mcpServers: [],
  issues: [],
};

export const effectiveChatHistory = <T extends ChatMessage>(messages: T[]) => {
  const summaryIndex = messages.findLastIndex(
    (item) => item.role === 'system' && item.content.startsWith(SUMMARY_MARKER),
  );
  return summaryIndex < 0 ? messages : messages.slice(summaryIndex);
};

export const estimateChatTokens = (
  messages: Array<Pick<ChatMessage, 'content'>>,
) =>
  messages.reduce(
    (total, item) =>
      total + Math.ceil(Array.from(item.content).length / 3.5) + 6,
    0,
  );

export const chatContextRatio = (
  messages: ChatMessage[],
  policy: ChatRuntimePolicy,
) =>
  Math.min(
    1,
    estimateChatTokens(effectiveChatHistory(messages)) /
      policy.contextWindowTokens,
  );

export const chatCompactionPlan = (
  messages: ChatMessage[],
  upcomingContent: string,
  policy: ChatRuntimePolicy,
) => {
  const effective = effectiveChatHistory(messages);
  const shouldCompact =
    effective.length > policy.compactKeepRecent &&
    estimateChatTokens([...effective, { content: upcomingContent }]) >=
      policy.contextWindowTokens * policy.compactThreshold;
  if (!shouldCompact) return undefined;
  return {
    summarized: effective.slice(0, -policy.compactKeepRecent),
    recent: effective.slice(-policy.compactKeepRecent),
  };
};

export const chatSummaryRequest = (messages: ChatMessage[]) => {
  const transcript = messages
    .map((item) => `${item.role.toUpperCase()}:\n${item.content}`)
    .join('\n\n');
  return [
    {
      role: 'system' as const,
      content:
        'Create a compact, factual continuation summary. Preserve the user goals, confirmed facts, constraints, key decisions, unfinished work, and tool results needed to continue. Do not invent information. Return only the summary.',
    },
    { role: 'user' as const, content: transcript },
  ];
};

export const skillSystemMessage = (
  skills: ChatSkillCapability[],
  enabledNames: string[],
): ChatRequestMessage[] => {
  const enabled = skills.filter((item) => enabledNames.includes(item.name));
  if (!enabled.length) return [];
  return [
    {
      role: 'system',
      content: [
        'The user enabled the following read-only Skills. Follow their instructions when relevant. They do not grant extra permissions.',
        ...enabled.map(
          (item) =>
            `<skill name="${item.name}">\n${item.content.trim()}\n</skill>`,
        ),
      ].join('\n\n'),
    },
  ];
};

export type MCPToolBinding = {
  server: string;
  tool: string;
  definition: ChatToolDefinition;
};

const modelToolName = (server: string, tool: string) =>
  `mcp__${server}__${tool}`.replace(/[^A-Za-z0-9_-]/g, '_').slice(0, 64);

export const mcpToolBindings = (
  servers: ChatMCPServerCapability[],
  enabledNames: string[],
) => {
  const bindings = new Map<string, MCPToolBinding>();
  for (const server of servers) {
    if (server.status !== 'connected' || !enabledNames.includes(server.name))
      continue;
    for (const tool of server.tools) {
      const name = modelToolName(server.name, tool.name);
      if (bindings.has(name)) continue;
      bindings.set(name, {
        server: server.name,
        tool: tool.name,
        definition: {
          type: 'function',
          function: {
            name,
            description: tool.description,
            parameters: tool.inputSchema || { type: 'object' },
          },
        },
      });
    }
  }
  return bindings;
};

const canRetry = (error: unknown) => {
  if (error instanceof GatewayRequestError) return error.retryable;
  return error instanceof TypeError;
};

const retryDelay = (
  error: unknown,
  attempt: number,
  policy: ChatRuntimePolicy,
) => {
  if (error instanceof GatewayRequestError && error.retryAfter !== undefined)
    return Math.min(error.retryAfter, policy.retryMaxMillis);
  const exponential = Math.min(
    policy.retryMaxMillis,
    policy.retryBaseMillis * 2 ** Math.max(0, attempt - 1),
  );
  return Math.round(exponential * (0.8 + Math.random() * 0.4));
};

export const retryChatRequest = async <T>(
  operation: (markActivity: () => void) => Promise<T>,
  policy: ChatRuntimePolicy,
  onRetry?: (nextAttempt: number, delay: number, error: unknown) => void,
) => {
  const attempts = Math.max(1, policy.maxRetries);
  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    let active = false;
    try {
      return await operation(() => {
        active = true;
      });
    } catch (error) {
      if (active || !canRetry(error) || attempt >= attempts) throw error;
      const delay = retryDelay(error, attempt, policy);
      onRetry?.(attempt + 1, delay, error);
      await new Promise((resolve) => window.setTimeout(resolve, delay));
    }
  }
  throw new Error('模型请求未完成');
};
