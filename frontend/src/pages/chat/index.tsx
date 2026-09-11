import {
  ApiOutlined,
  ArrowLeftOutlined,
  CheckOutlined,
  CompressOutlined,
  DeleteOutlined,
  DoubleLeftOutlined,
  DoubleRightOutlined,
  DownOutlined,
  DownloadOutlined,
  FileOutlined,
  FolderOpenOutlined,
  FolderOutlined,
  MenuOutlined,
  MessageOutlined,
  PlusOutlined,
  ReloadOutlined,
  RobotOutlined,
  SearchOutlined,
  SendOutlined,
  ThunderboltOutlined,
  ToolOutlined,
  UploadOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  Button,
  Empty,
  FloatButton,
  Input,
  Modal,
  Spin,
  Switch,
  Tooltip,
  theme,
  message as toast,
} from 'antd';
import type { CSSProperties, KeyboardEvent } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  chatCompactionPlan,
  chatContextRatio,
  chatSummaryRequest,
  effectiveChatHistory,
  emptyChatCapabilities,
  mcpToolBindings,
  retryChatRequest,
  SUMMARY_MARKER,
  skillSystemMessage,
} from '@/data-plane/chat-runtime';
import {
  type ChatMessage,
  type ChatRequestMessage,
  createChatCompletion,
  listGatewayModels,
} from '@/data-plane/client';
import {
  appendConversationMessage,
  type ChatCapabilities,
  type ChatWorkspaceListing,
  type ConversationSummary,
  callChatMCPTool,
  createConversation,
  deleteConversation,
  getChatCapabilities,
  getConversation,
  listChatWorkspace,
  listConversations,
  downloadChatWorkspaceFile,
  uploadChatWorkspaceFile,
  updateConversation,
} from '@/data-plane/conversations';
import { ChatMarkdown } from './chat_markdown_components';
import styles from './index.less';

type ConversationMessage = ChatMessage & {
  id: string;
  totalTokens?: number;
};

const generatedTitle = (content: string) => {
  const normalized = content.trim().replace(/\s+/g, ' ');
  const characters = Array.from(normalized);
  return characters.length > 48
    ? `${characters.slice(0, 48).join('')}…`
    : normalized;
};

const formatFileSize = (size: number) => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};

const ChatPage = () => {
  const intl = useIntl();
  const { token } = theme.useToken();
  const [models, setModels] = useState<string[]>([]);
  const [model, setModel] = useState<string>();
  const [loadingModels, setLoadingModels] = useState(true);
  const [modelPickerOpen, setModelPickerOpen] = useState(false);
  const [modelQuery, setModelQuery] = useState('');
  const [conversations, setConversations] = useState<ConversationSummary[]>([]);
  const [activeConversationID, setActiveConversationID] = useState<string>();
  const [loadingHistory, setLoadingHistory] = useState(true);
  const [loadingConversation, setLoadingConversation] = useState(false);
  const [conversationQuery, setConversationQuery] = useState('');
  const [historyOpen, setHistoryOpen] = useState(false);
  const [capabilitiesOpen, setCapabilitiesOpen] = useState(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(min-width: 1181px)').matches,
  );
  const [capabilities, setCapabilities] = useState<ChatCapabilities>(
    emptyChatCapabilities,
  );
  const [loadingCapabilities, setLoadingCapabilities] = useState(true);
  const [capabilityTab, setCapabilityTab] = useState<'capabilities' | 'files'>(
    'capabilities',
  );
  const [workspace, setWorkspace] = useState<ChatWorkspaceListing>();
  const [workspacePath, setWorkspacePath] = useState('');
  const [loadingWorkspace, setLoadingWorkspace] = useState(false);
  const [workspaceError, setWorkspaceError] = useState<string>();
  const [uploadingWorkspace, setUploadingWorkspace] = useState(false);
  const [downloadingPath, setDownloadingPath] = useState<string>();
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [selectedMCPServers, setSelectedMCPServers] = useState<string[]>([]);
  const [savingCapabilities, setSavingCapabilities] = useState(false);
  const [sending, setSending] = useState(false);
  const [runtimeStatus, setRuntimeStatus] = useState<string>();
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const listRef = useRef<HTMLDivElement>(null);
  const uploadInputRef = useRef<HTMLInputElement>(null);
  const scrollVersion =
    messages.length +
    Number(sending) +
    (messages[messages.length - 1]?.content.length || 0);
  const streamingMessageID = sending
    ? messages[messages.length - 1]?.id
    : undefined;

  const contextRatio = chatContextRatio(messages, capabilities.policy);
  const contextPercentage =
    contextRatio > 0 && contextRatio < 0.1
      ? `${(contextRatio * 100).toFixed(1)}%`
      : `${Math.round(contextRatio * 100)}%`;

  const loadModels = useCallback(async () => {
    setLoadingModels(true);
    try {
      const result = await listGatewayModels();
      const names = result.map((item) => item.id);
      setModels(names);
      setModel((current) =>
        current && names.includes(current) ? current : names[0],
      );
      if (!names.length) {
        toast.warning(intl.formatMessage({ id: 'chat.noModels' }));
      }
    } catch (error) {
      setModels([]);
      setModel(undefined);
      toast.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.loadModelsFailed' }),
      );
    } finally {
      setLoadingModels(false);
    }
  }, [intl]);

  const loadHistory = useCallback(async () => {
    setLoadingHistory(true);
    try {
      const result = await listConversations();
      setConversations(result);
      if (result[0]) {
        setLoadingConversation(true);
        const detail = await getConversation(result[0].id);
        setActiveConversationID(detail.id);
        setModel(detail.model);
        setSelectedSkills(detail.skills || []);
        setSelectedMCPServers(detail.mcpServers || []);
        setMessages(
          detail.messages.map((item) => ({
            id: item.id,
            role: item.role,
            content: item.content,
            totalTokens: item.totalTokens || undefined,
          })),
        );
      }
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.loadHistoryFailed' }),
      );
    } finally {
      setLoadingConversation(false);
      setLoadingHistory(false);
    }
  }, [intl]);

  const loadCapabilities = useCallback(async () => {
    setLoadingCapabilities(true);
    try {
      const result = await getChatCapabilities();
      setCapabilities(result);
    } catch (error) {
      setCapabilities(emptyChatCapabilities);
      toast.warning(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.capabilitiesFailed' }),
      );
    } finally {
      setLoadingCapabilities(false);
    }
  }, [intl]);

  const loadWorkspace = useCallback(
    async (path: string) => {
      setLoadingWorkspace(true);
      setWorkspaceError(undefined);
      setWorkspacePath(path);
      try {
        const result = await listChatWorkspace(path);
        setWorkspace(result);
        setWorkspacePath(result.path || '');
      } catch (error) {
        setWorkspaceError(
          error instanceof Error
            ? error.message
            : intl.formatMessage({ id: 'chat.workspaceLoadFailed' }),
        );
      } finally {
        setLoadingWorkspace(false);
      }
    },
    [intl],
  );

  useEffect(() => {
    void loadModels();
    void loadHistory();
    void loadCapabilities();
  }, [loadCapabilities, loadHistory, loadModels]);

  useEffect(() => {
    if (
      capabilitiesOpen &&
      capabilityTab === 'files' &&
      !workspace &&
      !workspaceError &&
      !loadingWorkspace
    ) {
      void loadWorkspace(workspacePath);
    }
  }, [
    capabilitiesOpen,
    capabilityTab,
    loadWorkspace,
    loadingWorkspace,
    workspace,
    workspaceError,
    workspacePath,
  ]);

  useEffect(() => {
    if (activeConversationID || messages.length) return;
    const defaults = capabilities.mcpServers
      .filter(
        (server) => server.status === 'connected' && server.defaultEnabled,
      )
      .map((server) => server.name);
    setSelectedMCPServers((current) =>
      current.length || !defaults.length ? current : defaults,
    );
  }, [activeConversationID, capabilities.mcpServers, messages.length]);

  useEffect(() => {
    const desktop = window.matchMedia('(min-width: 1181px)');
    const syncCapabilityPanel = (event: MediaQueryListEvent) =>
      setCapabilitiesOpen(event.matches);
    desktop.addEventListener('change', syncCapabilityPanel);
    return () => desktop.removeEventListener('change', syncCapabilityPanel);
  }, []);

  useEffect(() => {
    if (scrollVersion > 0) {
      listRef.current?.scrollTo({
        top: listRef.current.scrollHeight,
        behavior: 'smooth',
      });
    }
  }, [scrollVersion]);

  const activeConversation = conversations.find(
    (item) => item.id === activeConversationID,
  );

  const filteredModels = useMemo(() => {
    const query = modelQuery.trim().toLocaleLowerCase();
    if (!query) return models;
    return models.filter((name) => name.toLocaleLowerCase().includes(query));
  }, [modelQuery, models]);

  const filteredConversations = useMemo(() => {
    const query = conversationQuery.trim().toLocaleLowerCase();
    if (!query) return conversations;
    return conversations.filter((item) =>
      (item.title || intl.formatMessage({ id: 'chat.newConversation' }))
        .toLocaleLowerCase()
        .includes(query),
    );
  }, [conversationQuery, conversations, intl]);

  const promoteConversation = (
    id: string,
    patch?: Partial<ConversationSummary>,
  ) => {
    setConversations((current) => {
      const existing = current.find((item) => item.id === id);
      if (!existing) return current;
      const next = {
        ...existing,
        ...patch,
        updatedAt: patch?.updatedAt || new Date().toISOString(),
      };
      return [next, ...current.filter((item) => item.id !== id)];
    });
  };

  const openConversation = async (id: string) => {
    if (sending || loadingConversation || id === activeConversationID) return;
    setLoadingConversation(true);
    setHistoryOpen(false);
    try {
      const detail = await getConversation(id);
      setActiveConversationID(detail.id);
      setModel(detail.model);
      setSelectedSkills(detail.skills || []);
      setSelectedMCPServers(detail.mcpServers || []);
      setMessages(
        detail.messages.map((item) => ({
          id: item.id,
          role: item.role,
          content: item.content,
          totalTokens: item.totalTokens || undefined,
        })),
      );
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.loadConversationFailed' }),
      );
    } finally {
      setLoadingConversation(false);
    }
  };

  const startNewConversation = () => {
    if (sending || loadingConversation) return;
    setActiveConversationID(undefined);
    setMessages([]);
    setInput('');
    setSelectedSkills([]);
    setSelectedMCPServers(
      capabilities.mcpServers
        .filter(
          (server) => server.status === 'connected' && server.defaultEnabled,
        )
        .map((server) => server.name),
    );
    setHistoryOpen(false);
  };

  const selectModel = async (name: string) => {
    const previous = model;
    setModel(name);
    setModelPickerOpen(false);
    setModelQuery('');
    if (!activeConversationID) return;
    try {
      const updated = await updateConversation(activeConversationID, {
        model: name,
        skills: selectedSkills,
        mcpServers: selectedMCPServers,
      });
      promoteConversation(updated.id, updated);
    } catch (error) {
      setModel(previous);
      toast.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.updateConversationFailed' }),
      );
    }
  };

  const confirmDeleteConversation = (conversation: ConversationSummary) => {
    Modal.confirm({
      title: intl.formatMessage({ id: 'chat.deleteConversationTitle' }),
      content: intl.formatMessage({ id: 'chat.deleteConversationHint' }),
      okText: intl.formatMessage({ id: 'common.delete' }),
      cancelText: intl.formatMessage({ id: 'common.cancel' }),
      okButtonProps: { danger: true },
      onOk: async () => {
        await deleteConversation(conversation.id);
        setConversations((current) =>
          current.filter((item) => item.id !== conversation.id),
        );
        if (activeConversationID === conversation.id) {
          setActiveConversationID(undefined);
          setMessages([]);
          setSelectedSkills([]);
          setSelectedMCPServers([]);
        }
      },
    });
  };

  const changeCapabilities = async (
    nextSkills: string[],
    nextMCPServers: string[],
  ) => {
    if (!model || sending || loadingConversation || savingCapabilities) return;
    const previousSkills = selectedSkills;
    const previousMCPServers = selectedMCPServers;
    setSelectedSkills(nextSkills);
    setSelectedMCPServers(nextMCPServers);
    if (!activeConversationID) return;
    setSavingCapabilities(true);
    try {
      const updated = await updateConversation(activeConversationID, {
        model,
        skills: nextSkills,
        mcpServers: nextMCPServers,
      });
      promoteConversation(updated.id, updated);
    } catch (error) {
      setSelectedSkills(previousSkills);
      setSelectedMCPServers(previousMCPServers);
      toast.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.updateConversationFailed' }),
      );
    } finally {
      setSavingCapabilities(false);
    }
  };

  const persistAssistantMessage = async (
    conversationID: string,
    content: string,
    totalTokens?: number,
  ) => {
    try {
      await appendConversationMessage(conversationID, {
        role: 'assistant',
        content,
        totalTokens,
      });
      promoteConversation(conversationID);
    } catch {
      toast.warning(intl.formatMessage({ id: 'chat.saveReplyFailed' }));
    }
  };

  const send = async () => {
    const content = input.trim();
    if (!content || !model || sending || loadingCapabilities) return;
    const userMessage: ConversationMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
    };
    const assistantID = crypto.randomUUID();
    setMessages((current) => [
      ...current,
      userMessage,
      { id: assistantID, role: 'assistant', content: '' },
    ]);
    setInput('');
    setSending(true);
    setRuntimeStatus(intl.formatMessage({ id: 'chat.statusPreparing' }));

    let conversationID = activeConversationID;
    let userPersisted = false;
    let assistantContent = '';
    try {
      if (!conversationID) {
        const created = await createConversation(
          model,
          selectedSkills,
          selectedMCPServers,
        );
        conversationID = created.id;
        setActiveConversationID(created.id);
        setConversations((current) => [created, ...current]);
      }

      const storedHistory = messages.map(({ role, content: text }) => ({
        role,
        content: text,
      }));
      const compaction = chatCompactionPlan(
        storedHistory,
        content,
        capabilities.policy,
      );
      let effectiveHistory = effectiveChatHistory(storedHistory);
      if (compaction) {
        setRuntimeStatus(intl.formatMessage({ id: 'chat.statusCompacting' }));
        try {
          const summary = await retryChatRequest(
            (markActivity) =>
              createChatCompletion(
                model,
                chatSummaryRequest(compaction.summarized),
                {
                  onActivity: markActivity,
                },
              ),
            capabilities.policy,
            (attempt, delay) =>
              setRuntimeStatus(
                intl.formatMessage(
                  { id: 'chat.statusRetrying' },
                  {
                    attempt,
                    max: capabilities.policy.maxRetries,
                    seconds: Math.max(1, Math.ceil(delay / 1000)),
                  },
                ),
              ),
          );
          if (summary.content.trim()) {
            const summaryContent = `${SUMMARY_MARKER}\n${summary.content.trim()}`;
            const savedSummary = await appendConversationMessage(
              conversationID,
              { role: 'system', content: summaryContent },
            );
            const summaryMessage: ConversationMessage = {
              id: savedSummary.id,
              role: 'system',
              content: summaryContent,
              totalTokens: savedSummary.totalTokens || undefined,
            };
            setMessages((current) => {
              const userIndex = current.findIndex(
                (item) => item.id === userMessage.id,
              );
              if (userIndex < 0) return [...current, summaryMessage];
              return [
                ...current.slice(0, userIndex),
                summaryMessage,
                ...current.slice(userIndex),
              ];
            });
            effectiveHistory = [
              { role: 'system', content: summaryContent },
              ...compaction.recent,
            ];
          }
        } catch {
          toast.warning(intl.formatMessage({ id: 'chat.compactionFailed' }));
        }
      }

      await appendConversationMessage(conversationID, {
        role: 'user',
        content,
      });
      userPersisted = true;
      promoteConversation(conversationID, {
        title: activeConversation?.title || generatedTitle(content),
      });
      setRuntimeStatus(intl.formatMessage({ id: 'chat.statusRequesting' }));

      const bindings = mcpToolBindings(
        capabilities.mcpServers,
        selectedMCPServers,
      );
      const tools = [...bindings.values()].map((item) => item.definition);
      const requestMessages: ChatRequestMessage[] = [
        ...skillSystemMessage(capabilities.skills, selectedSkills),
        ...(selectedMCPServers.includes('builtin')
          ? [
              {
                role: 'system' as const,
                content:
                  'The builtin file tools are securely scoped by the server to the current signed-in user personal workspace. Use only workspace-relative paths; never attempt parent paths, another user directory, or a physical server path. When delivering a file, write the finished artifact to this workspace and mention its relative path so the user can download it from the Files tab.',
              },
            ]
          : []),
        ...(tools.length
          ? [
              {
                role: 'system' as const,
                content:
                  'Tools listed on this request are real capabilities in the Token Router service environment. When the user asks to inspect or change that environment, use the relevant tool and report its actual result. Never replace an available tool call with simulated output or claim that you cannot execute it. For an unfamiliar CLI, call discover_commands first; use command for installed CLI programs such as kubectl.',
              },
            ]
          : []),
        ...effectiveHistory,
        { role: 'user', content },
      ];
      let totalTokens = 0;
      let completed = false;

      for (
        let round = 0;
        round <= capabilities.policy.maxToolRounds;
        round += 1
      ) {
        const isFinalizationRound =
          tools.length > 0 && round === capabilities.policy.maxToolRounds;
        if (isFinalizationRound) {
          requestMessages.push({
            role: 'system',
            content:
              'The tool-call safety limit has been reached. Do not request more tools. Complete the response using the tool results already available, and clearly state any work that remains unfinished.',
          });
        }
        const result = await retryChatRequest(
          (markActivity) => {
            setRuntimeStatus(
              intl.formatMessage({ id: 'chat.statusRequesting' }),
            );
            return createChatCompletion(model, requestMessages, {
              conversationID,
              toolChoice: tools.length
                ? isFinalizationRound
                  ? 'none'
                  : 'auto'
                : undefined,
              onActivity: markActivity,
              onContent: (streamed) => {
                assistantContent = streamed;
                setMessages((current) =>
                  current.map((item) =>
                    item.id === assistantID
                      ? { ...item, content: streamed }
                      : item,
                  ),
                );
              },
            });
          },
          capabilities.policy,
          (attempt, delay) =>
            setRuntimeStatus(
              intl.formatMessage(
                { id: 'chat.statusRetrying' },
                {
                  attempt,
                  max: capabilities.policy.maxRetries,
                  seconds: Math.max(1, Math.ceil(delay / 1000)),
                },
              ),
            ),
        );
        totalTokens += result.totalTokens || 0;
        assistantContent = result.content;
        if (!result.toolCalls.length) {
          completed = true;
          break;
        }
        if (isFinalizationRound) break;

        requestMessages.push({
          role: 'assistant',
          content: result.content || null,
          tool_calls: result.toolCalls,
        });
        for (const toolCall of result.toolCalls) {
          const binding = bindings.get(toolCall.function.name);
          setRuntimeStatus(
            intl.formatMessage(
              { id: 'chat.statusTool' },
              { tool: binding?.tool || toolCall.function.name },
            ),
          );
          let toolContent: string;
          if (!binding) {
            toolContent = JSON.stringify({
              error: 'The requested MCP tool is not available',
            });
          } else {
            try {
              const args = JSON.parse(
                toolCall.function.arguments || '{}',
              ) as Record<string, unknown>;
              const toolResult = await callChatMCPTool(
                binding.server,
                binding.tool,
                args,
              );
              toolContent =
                toolResult.content ||
                JSON.stringify({ isError: toolResult.isError });
            } catch (error) {
              toolContent = JSON.stringify({
                error:
                  error instanceof Error
                    ? error.message
                    : 'MCP tool call failed',
              });
            }
          }
          requestMessages.push({
            role: 'tool',
            tool_call_id: toolCall.id || crypto.randomUUID(),
            name: toolCall.function.name,
            content: toolContent,
          });
        }
        assistantContent = '';
        setMessages((current) =>
          current.map((item) =>
            item.id === assistantID ? { ...item, content: '' } : item,
          ),
        );
      }

      if (!completed) {
        throw new Error(intl.formatMessage({ id: 'chat.toolRoundLimit' }));
      }
      const finalContent =
        assistantContent || intl.formatMessage({ id: 'chat.emptyResponse' });
      setMessages((current) =>
        current.map((item) =>
          item.id === assistantID
            ? {
                ...item,
                content: finalContent,
                totalTokens: totalTokens || undefined,
              }
            : item,
        ),
      );
      setRuntimeStatus(intl.formatMessage({ id: 'chat.statusSaving' }));
      await persistAssistantMessage(
        conversationID,
        finalContent,
        totalTokens || undefined,
      );
      if (capabilityTab === 'files') {
        void loadWorkspace(workspacePath);
      }
    } catch (error) {
      const detail =
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.requestFailed' });
      const failureMessage = intl.formatMessage(
        { id: 'chat.callFailed' },
        { detail },
      );
      const errorContent = assistantContent
        ? `${assistantContent}\n\n---\n${failureMessage}`
        : failureMessage;
      setMessages((current) =>
        current.map((item) =>
          item.id === assistantID ? { ...item, content: errorContent } : item,
        ),
      );
      if (conversationID && userPersisted) {
        await persistAssistantMessage(conversationID, errorContent);
      }
    } finally {
      setSending(false);
      setRuntimeStatus(undefined);
    }
  };

  const onComposerKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      void send();
    }
  };

  const openWorkspaceDirectory = (path: string) => {
    void loadWorkspace(path);
  };

  const openWorkspaceParent = () => {
    const parts = workspacePath.split('/').filter(Boolean);
    parts.pop();
    void loadWorkspace(parts.join('/'));
  };

  const uploadWorkspaceFile = async (file?: File) => {
    if (!file) return;
    setUploadingWorkspace(true);
    try {
      await uploadChatWorkspaceFile(workspacePath, file);
      toast.success(intl.formatMessage({ id: 'chat.workspaceUploadSucceeded' }));
      await loadWorkspace(workspacePath);
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.workspaceUploadFailed' }),
      );
    } finally {
      setUploadingWorkspace(false);
      if (uploadInputRef.current) uploadInputRef.current.value = '';
    }
  };

  const downloadWorkspaceFile = async (path: string, name: string) => {
    setDownloadingPath(path);
    try {
      const blob = await downloadChatWorkspaceFile(path);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = name;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 0);
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.workspaceDownloadFailed' }),
      );
    } finally {
      setDownloadingPath(undefined);
    }
  };

  return (
    <main
      className={`${styles.chatPage} ${capabilitiesOpen ? '' : styles.chatPageCapabilitiesCollapsed}`}
    >
      <aside
        className={`${styles.historyPanel} ${historyOpen ? styles.historyPanelOpen : ''}`}
      >
        <div className={styles.historyHeader}>
          <span className={styles.historyBrand}>
            <MessageOutlined />
            {intl.formatMessage({ id: 'chat.history' })}
          </span>
          <Tooltip
            color={token.colorPrimary}
            title={intl.formatMessage({ id: 'chat.newConversation' })}
          >
            <Button
              aria-label={intl.formatMessage({ id: 'chat.newConversation' })}
              disabled={sending || loadingConversation}
              icon={<PlusOutlined />}
              onClick={startNewConversation}
              shape="circle"
              type="text"
            />
          </Tooltip>
        </div>
        <Input
          allowClear
          className={styles.conversationSearch}
          onChange={(event) => setConversationQuery(event.target.value)}
          placeholder={intl.formatMessage({ id: 'chat.searchConversation' })}
          prefix={<SearchOutlined />}
          value={conversationQuery}
        />
        <div className={styles.historySectionLabel}>
          {intl.formatMessage({ id: 'chat.recentConversations' })}
        </div>
        <div className={styles.historyList}>
          {loadingHistory ? (
            <div className={styles.historyLoading}>
              <Spin size="small" />
            </div>
          ) : filteredConversations.length ? (
            filteredConversations.map((conversation) => (
              <div
                className={`${styles.historyItem} ${activeConversationID === conversation.id ? styles.historyItemActive : ''}`}
                key={conversation.id}
              >
                <button
                  className={styles.historyItemMain}
                  disabled={sending || loadingConversation}
                  onClick={() => void openConversation(conversation.id)}
                  type="button"
                >
                  <span className={styles.historyItemTitle}>
                    {conversation.title ||
                      intl.formatMessage({ id: 'chat.newConversation' })}
                  </span>
                  <span className={styles.historyItemMeta}>
                    {conversation.model} ·{' '}
                    {new Date(conversation.updatedAt).toLocaleDateString(
                      intl.locale,
                      { month: 'short', day: 'numeric' },
                    )}
                  </span>
                </button>
                <Tooltip
                  color={token.colorPrimary}
                  title={intl.formatMessage({ id: 'common.delete' })}
                >
                  <Button
                    aria-label={intl.formatMessage({ id: 'common.delete' })}
                    className={styles.historyDelete}
                    disabled={sending || loadingConversation}
                    icon={<DeleteOutlined />}
                    onClick={() => confirmDeleteConversation(conversation)}
                    size="small"
                    type="text"
                  />
                </Tooltip>
              </div>
            ))
          ) : (
            <Empty
              className={styles.historyEmpty}
              description={intl.formatMessage({ id: 'chat.noConversations' })}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          )}
        </div>
      </aside>

      {historyOpen ? (
        <button
          aria-label={intl.formatMessage({ id: 'common.close' })}
          className={styles.historyBackdrop}
          onClick={() => setHistoryOpen(false)}
          type="button"
        />
      ) : null}

      <section className={styles.conversation}>
        <header className={styles.conversationHeader}>
          <div className={styles.headerTitle}>
            <Button
              aria-label={intl.formatMessage({ id: 'chat.history' })}
              className={styles.mobileHistoryButton}
              icon={<MenuOutlined />}
              onClick={() => setHistoryOpen(true)}
              type="text"
            />
            <span>
              <h1>
                {activeConversation?.title ||
                  intl.formatMessage({ id: 'chat.newConversation' })}
              </h1>
              <small>
                {model || intl.formatMessage({ id: 'chat.selectModel' })}
              </small>
            </span>
          </div>
          <div className={styles.headerActions}>
            <Button
              className={styles.modelPickerButton}
              disabled={sending || loadingConversation}
              icon={<RobotOutlined />}
              loading={loadingModels}
              onClick={() => setModelPickerOpen(true)}
            >
              <span className={styles.modelPickerLabel}>
                {model || intl.formatMessage({ id: 'chat.selectModel' })}
              </span>
              <DownOutlined className={styles.modelPickerChevron} />
            </Button>
          </div>
        </header>

        <div className={styles.messageList} ref={listRef}>
          {loadingConversation ? (
            <div className={styles.conversationLoading}>
              <Spin />
            </div>
          ) : messages.length ? (
            messages.map((item) =>
              item.role === 'system' &&
              item.content.startsWith(SUMMARY_MARKER) ? (
                <article className={styles.summaryEvent} key={item.id}>
                  <span>
                    <CompressOutlined />
                    {intl.formatMessage({ id: 'chat.summaryEvent' })}
                  </span>
                  <small>
                    {intl.formatMessage({ id: 'chat.summaryEventHint' })}
                  </small>
                </article>
              ) : (
                <article
                  className={`${styles.message} ${item.role === 'user' ? styles.user : ''}`}
                  key={item.id}
                >
                  <span className={styles.avatar}>
                    {item.role === 'user' ? (
                        <UserOutlined color={ token.colorPrimary} />
                    ) : (
                      <RobotOutlined />
                    )}
                  </span>
                  <div className={styles.messageBody}>
                    <span className={styles.messageLabel}>
                      {item.role === 'user'
                        ? intl.formatMessage({ id: 'chat.you' })
                        : intl.formatMessage({ id: 'chat.assistant' })}
                    </span>
                    <div
                      aria-busy={item.id === streamingMessageID}
                      className={styles.bubble}
                    >
                      <ChatMarkdown
                        content={item.content}
                        streaming={item.id === streamingMessageID}
                      />
                    </div>
                    {item.totalTokens ? (
                      <span className={styles.messageMeta}>
                        {item.totalTokens} TOKENS
                      </span>
                    ) : null}
                  </div>
                </article>
              ),
            )
          ) : (
            <div className={styles.welcomeState}>
              <span className={styles.welcomeGlyph}>
                <RobotOutlined />
              </span>
              <h2>{intl.formatMessage({ id: 'chat.emptyTitle' })}</h2>
              <p>{intl.formatMessage({ id: 'chat.emptyHint' })}</p>
            </div>
          )}
        </div>

        <footer className={styles.composerWrap}>
          {runtimeStatus ? (
            <div className={styles.runtimeStatus} role="status">
              <span />
              {runtimeStatus}
            </div>
          ) : null}
          <div className={styles.composer}>
            <Input.TextArea
              autoSize={{ minRows: 1, maxRows: 7 }}
              disabled={
                !model || loadingConversation || loadingCapabilities || sending
              }
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={onComposerKeyDown}
              placeholder={intl.formatMessage({
                id: model ? 'chat.input' : 'chat.selectModelFirst',
              })}
              value={input}
            />
            <Button
              aria-label={intl.formatMessage({ id: 'chat.send' })}
              disabled={
                !model ||
                !input.trim() ||
                loadingConversation ||
                loadingCapabilities ||
                sending
              }
              icon={<SendOutlined />}
              loading={sending}
              onClick={() => void send()}
              shape="circle"
              size="large"
              type="primary"
            />
          </div>
          <span className={styles.composerHint}>
            {intl.formatMessage({ id: 'chat.hint' })}
          </span>
        </footer>
      </section>

      {capabilitiesOpen ? (
        <button
          aria-label={intl.formatMessage({ id: 'common.close' })}
          className={styles.capabilityBackdrop}
          onClick={() => setCapabilitiesOpen(false)}
          type="button"
        />
      ) : null}

      <aside
        aria-hidden={!capabilitiesOpen}
        className={`${styles.capabilityPanel} ${
          capabilitiesOpen
            ? styles.capabilityPanelOpen
            : styles.capabilityPanelCollapsed
        }`}
        inert={!capabilitiesOpen}
      >
        <div className={styles.capabilityHeader}>
          <span>
            <small>AGENT RUNTIME</small>
            <strong>{intl.formatMessage({ id: 'chat.capabilities' })}</strong>
          </span>
          <Button
            onClick={() => setCapabilitiesOpen(false)}
            size="small"
            type="text"
          >
            {intl.formatMessage({ id: 'common.close' })}
          </Button>
        </div>

        <div className={styles.capabilityTabs} role="tablist">
          <button
            aria-selected={capabilityTab === 'capabilities'}
            className={
              capabilityTab === 'capabilities' ? styles.capabilityTabActive : ''
            }
            onClick={() => setCapabilityTab('capabilities')}
            role="tab"
            type="button"
          >
            <ToolOutlined />
            {intl.formatMessage({ id: 'chat.capabilityTab' })}
          </button>
          <button
            aria-selected={capabilityTab === 'files'}
            className={capabilityTab === 'files' ? styles.capabilityTabActive : ''}
            onClick={() => {
              setCapabilityTab('files');
              void loadWorkspace(workspacePath);
            }}
            role="tab"
            type="button"
          >
            <FolderOpenOutlined />
            {intl.formatMessage({ id: 'chat.filesTab' })}
          </button>
        </div>

        {capabilityTab === 'capabilities' ? loadingCapabilities ? (
          <div className={styles.capabilityLoading}>
            <Spin size="small" />
          </div>
        ) : (
          <div className={styles.capabilityBody}>
            <section className={styles.contextCard}>
              <div className={styles.contextGauge}>
                <span style={{ '--context': contextRatio } as CSSProperties} />
                <strong>{contextPercentage}</strong>
              </div>
              <div>
                <strong>{intl.formatMessage({ id: 'chat.context' })}</strong>
                <small>
                  {intl.formatMessage(
                    { id: 'chat.contextHint' },
                    {
                      threshold: Math.round(
                        capabilities.policy.compactThreshold * 100,
                      ),
                    },
                  )}
                </small>
              </div>
            </section>

            <div className={styles.automationGrid}>
              <span>
                <CompressOutlined />
                <strong>
                  {intl.formatMessage({ id: 'chat.autoCompact' })}
                </strong>
                <small>
                  {capabilities.policy.compactKeepRecent}{' '}
                  {intl.formatMessage({ id: 'chat.messagesKept' })}
                </small>
              </span>
              <span>
                <ThunderboltOutlined />
                <strong>{intl.formatMessage({ id: 'chat.autoRetry' })}</strong>
                <small>
                  {capabilities.policy.maxRetries}{' '}
                  {intl.formatMessage({ id: 'chat.attempts' })}
                </small>
              </span>
              <span className={styles.toolRoundPolicy}>
                <ToolOutlined />
                <strong>{intl.formatMessage({ id: 'chat.toolRounds' })}</strong>
                <small>
                  {intl.formatMessage(
                    { id: 'chat.toolRoundsHint' },
                    { rounds: capabilities.policy.maxToolRounds },
                  )}
                </small>
              </span>
            </div>

            <section className={styles.capabilitySection}>
              <header>
                <span>SKILLS</span>
                <small>{capabilities.skills.length}</small>
              </header>
              {capabilities.skills.length ? (
                capabilities.skills.map((skill) => (
                  <div className={styles.capabilityItem} key={skill.name}>
                    <span>
                      <strong>{skill.name}</strong>
                      <small>{skill.description}</small>
                    </span>
                    <Switch
                      checked={selectedSkills.includes(skill.name)}
                      disabled={
                        sending ||
                        loadingConversation ||
                        savingCapabilities ||
                        !model
                      }
                      onChange={(checked) =>
                        void changeCapabilities(
                          checked
                            ? [...selectedSkills, skill.name]
                            : selectedSkills.filter(
                                (name) => name !== skill.name,
                              ),
                          selectedMCPServers,
                        )
                      }
                      size="small"
                    />
                  </div>
                ))
              ) : (
                <p className={styles.capabilityEmpty}>
                  {intl.formatMessage({ id: 'chat.noSkills' })}
                </p>
              )}
            </section>

            <section className={styles.capabilitySection}>
              <header>
                <span>Tools / MCP</span>
                <small>{capabilities.mcpServers.length}</small>
              </header>
              {capabilities.mcpServers.length ? (
                capabilities.mcpServers.map((server) => (
                  <div className={styles.capabilityItem} key={server.name}>
                    <ApiOutlined />
                    <span>
                      <strong>
                        {server.name}
                        <i
                          className={
                            server.status === 'connected'
                              ? styles.connectedDot
                              : styles.failedDot
                          }
                        />
                      </strong>
                      <small>
                        {server.status === 'connected'
                          ? intl.formatMessage(
                              { id: 'chat.toolCount' },
                              { count: server.tools.length },
                            )
                          : server.error ||
                            intl.formatMessage({ id: 'chat.mcpUnavailable' })}
                      </small>
                    </span>
                    <Switch
                      checked={selectedMCPServers.includes(server.name)}
                      disabled={
                        sending ||
                        loadingConversation ||
                        savingCapabilities ||
                        !model ||
                        server.status !== 'connected'
                      }
                      onChange={(checked) =>
                        void changeCapabilities(
                          selectedSkills,
                          checked
                            ? [...selectedMCPServers, server.name]
                            : selectedMCPServers.filter(
                                (name) => name !== server.name,
                              ),
                        )
                      }
                      size="small"
                    />
                  </div>
                ))
              ) : (
                <p className={styles.capabilityEmpty}>
                  {intl.formatMessage({ id: 'chat.noMCP' })}
                </p>
              )}
            </section>

            {capabilities.issues.length ? (
              <div className={styles.capabilityIssues}>
                {capabilities.issues.join(' · ')}
              </div>
            ) : null}
            <Button
              block
              icon={<ReloadOutlined />}
              onClick={() => void loadCapabilities()}
              size="small"
              type="text"
            >
              {intl.formatMessage({ id: 'chat.refreshCapabilities' })}
            </Button>
          </div>
        ) : (
          <div className={styles.workspaceBody}>
            <div className={styles.workspaceToolbar}>
              <Button
                aria-label={intl.formatMessage({ id: 'chat.workspaceParent' })}
                disabled={!workspacePath || loadingWorkspace}
                icon={<ArrowLeftOutlined />}
                onClick={openWorkspaceParent}
                size="small"
                type="text"
              />
              <div className={styles.workspaceBreadcrumbs}>
                <button
                  onClick={() => openWorkspaceDirectory('')}
                  type="button"
                >
                  {intl.formatMessage({ id: 'chat.workspaceRoot' })}
                </button>
                {workspacePath
                  .split('/')
                  .filter(Boolean)
                  .map((part, index, parts) => (
                    <span key={parts.slice(0, index + 1).join('/')}>
                      <i>/</i>
                      <button
                        onClick={() =>
                          openWorkspaceDirectory(parts.slice(0, index + 1).join('/'))
                        }
                        type="button"
                      >
                        {part}
                      </button>
                    </span>
                  ))}
              </div>
              <Tooltip
                color={token.colorPrimary}
                title={intl.formatMessage({ id: 'common.refresh' })}
              >
                <Button
                  aria-label={intl.formatMessage({ id: 'common.refresh' })}
                  icon={<ReloadOutlined />}
                  loading={loadingWorkspace}
                  onClick={() => void loadWorkspace(workspacePath)}
                  size="small"
                  type="text"
                />
              </Tooltip>
            </div>

            <div className={styles.workspaceIntro}>
              <span>
                <strong>{intl.formatMessage({ id: 'chat.personalWorkspace' })}</strong>
                <small>
                  {intl.formatMessage({ id: 'chat.personalWorkspaceHint' })}
                </small>
              </span>
              <Button
                icon={<UploadOutlined />}
                loading={uploadingWorkspace}
                onClick={() => uploadInputRef.current?.click()}
                size="small"
                type="primary"
              >
                {intl.formatMessage({ id: 'chat.uploadFile' })}
              </Button>
              <input
                hidden
                onChange={(event) =>
                  void uploadWorkspaceFile(event.target.files?.[0])
                }
                ref={uploadInputRef}
                type="file"
              />
            </div>

            {workspaceError ? (
              <div className={styles.workspaceError}>
                <FolderOutlined />
                <strong>{intl.formatMessage({ id: 'chat.workspaceUnavailable' })}</strong>
                <small>{workspaceError}</small>
                <Button
                  onClick={() => void loadWorkspace(workspacePath)}
                  size="small"
                >
                  {intl.formatMessage({ id: 'chat.retry' })}
                </Button>
              </div>
            ) : loadingWorkspace && !workspace ? (
              <div className={styles.capabilityLoading}>
                <Spin size="small" />
              </div>
            ) : workspace?.entries.length ? (
              <div className={styles.workspaceList}>
                {workspace.entries.map((entry) => (
                  <div className={styles.workspaceItem} key={entry.path}>
                    <button
                      disabled={entry.type !== 'directory'}
                      onClick={() => openWorkspaceDirectory(entry.path)}
                      type="button"
                    >
                      <span
                        className={
                          entry.type === 'directory'
                            ? styles.workspaceFolderIcon
                            : styles.workspaceFileIcon
                        }
                      >
                        {entry.type === 'directory' ? (
                          <FolderOutlined />
                        ) : (
                          <FileOutlined />
                        )}
                      </span>
                      <span>
                        <strong>{entry.name}</strong>
                        <small>
                          {entry.type === 'directory'
                            ? intl.formatMessage({ id: 'chat.folder' })
                            : entry.type === 'symlink'
                              ? intl.formatMessage({ id: 'chat.symbolicLink' })
                              : `${formatFileSize(entry.size)} · ${new Date(
                                  entry.updatedAt,
                                ).toLocaleDateString(intl.locale, {
                                  month: 'short',
                                  day: 'numeric',
                                })}`}
                        </small>
                      </span>
                    </button>
                    {entry.type === 'file' ? (
                      <Tooltip
                        color={token.colorPrimary}
                        title={intl.formatMessage({ id: 'chat.downloadFile' })}
                      >
                        <Button
                          aria-label={intl.formatMessage({ id: 'chat.downloadFile' })}
                          icon={<DownloadOutlined />}
                          loading={downloadingPath === entry.path}
                          onClick={() =>
                            void downloadWorkspaceFile(entry.path, entry.name)
                          }
                          size="small"
                          type="text"
                        />
                      </Tooltip>
                    ) : null}
                  </div>
                ))}
              </div>
            ) : (
              <Empty
                className={styles.workspaceEmpty}
                description={intl.formatMessage({ id: 'chat.workspaceEmpty' })}
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            )}
            {workspace ? (
              <small className={styles.workspaceLimit}>
                {intl.formatMessage(
                  { id: 'chat.workspaceUploadLimit' },
                  { size: formatFileSize(workspace.maxUploadBytes) },
                )}
              </small>
            ) : null}
          </div>
        )}
      </aside>

      <FloatButton
        aria-expanded={capabilitiesOpen}
        aria-label={intl.formatMessage({ id: 'chat.toggleCapabilities' })}
        badge={{ count: selectedSkills.length + selectedMCPServers.length }}
        className={`${styles.capabilityFloatButton} ${
          capabilitiesOpen ? styles.capabilityFloatButtonActive : ''
        }`}
        icon={
          capabilitiesOpen ? <DoubleRightOutlined /> : <DoubleLeftOutlined />
        }
        onClick={() => setCapabilitiesOpen((current) => !current)}
        tooltip={{
          color: token.colorPrimary,
          title: intl.formatMessage({ id: 'chat.toggleCapabilities' }),
        }}
      />

      <Modal
        centered
        footer={null}
        onCancel={() => {
          setModelPickerOpen(false);
          setModelQuery('');
        }}
        open={modelPickerOpen}
        title={intl.formatMessage({ id: 'chat.selectModelTitle' })}
        width={520}
      >
        <p className={styles.modelPickerDescription}>
          {intl.formatMessage({ id: 'chat.selectModelDescription' })}
        </p>
        <Input
          allowClear
          className={styles.modelSearch}
          onChange={(event) => setModelQuery(event.target.value)}
          placeholder={intl.formatMessage({ id: 'chat.searchModel' })}
          prefix={<SearchOutlined />}
          value={modelQuery}
        />
        {loadingModels ? (
          <div className={styles.modelPickerLoading}>
            <Spin />
          </div>
        ) : filteredModels.length ? (
          <div className={styles.modelOptions}>
            {filteredModels.map((name) => (
              <button
                aria-pressed={model === name}
                className={`${styles.modelOption} ${model === name ? styles.modelOptionSelected : ''}`}
                key={name}
                onClick={() => void selectModel(name)}
                type="button"
              >
                <span className={styles.modelOptionGlyph}>
                  <RobotOutlined />
                </span>
                <span className={styles.modelOptionText}>
                  <strong>{name}</strong>
                  <small>Token Router</small>
                </span>
                {model === name ? <CheckOutlined /> : null}
              </button>
            ))}
          </div>
        ) : (
          <Empty
            description={intl.formatMessage({ id: 'chat.noMatchingModels' })}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            {!modelQuery ? (
              <Button
                icon={<ReloadOutlined />}
                onClick={() => void loadModels()}
              >
                {intl.formatMessage({ id: 'common.refresh' })}
              </Button>
            ) : null}
          </Empty>
        )}
      </Modal>
    </main>
  );
};

export default ChatPage;
