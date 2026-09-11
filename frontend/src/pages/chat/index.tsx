import {
  CheckOutlined,
  DeleteOutlined,
  DownOutlined,
  MenuOutlined,
  MessageOutlined,
  PlusOutlined,
  ReloadOutlined,
  RobotOutlined,
  SearchOutlined,
  SendOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  Button,
  Empty,
  Input,
  message as toast,
  Modal,
  Spin,
  Tooltip,
} from 'antd';
import { useIntl } from '@umijs/max';
import type { KeyboardEvent } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  appendConversationMessage,
  createConversation,
  deleteConversation,
  getConversation,
  listConversations,
  type ConversationSummary,
  updateConversation,
} from '@/data-plane/conversations';
import {
  createChatCompletion,
  listGatewayModels,
  type ChatMessage,
} from '@/data-plane/client';
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

const ChatPage = () => {
  const intl = useIntl();
  const [models, setModels] = useState<string[]>([]);
  const [model, setModel] = useState<string>();
  const [loadingModels, setLoadingModels] = useState(true);
  const [modelPickerOpen, setModelPickerOpen] = useState(false);
  const [modelQuery, setModelQuery] = useState('');
  const [conversations, setConversations] = useState<ConversationSummary[]>(
    [],
  );
  const [activeConversationID, setActiveConversationID] = useState<string>();
  const [loadingHistory, setLoadingHistory] = useState(true);
  const [loadingConversation, setLoadingConversation] = useState(false);
  const [conversationQuery, setConversationQuery] = useState('');
  const [historyOpen, setHistoryOpen] = useState(false);
  const [sending, setSending] = useState(false);
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const listRef = useRef<HTMLDivElement>(null);
  const scrollVersion =
    messages.length +
    Number(sending) +
    (messages[messages.length - 1]?.content.length || 0);
  const streamingMessageID = sending
    ? messages[messages.length - 1]?.id
    : undefined;

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

  useEffect(() => {
    void loadModels();
    void loadHistory();
  }, [loadHistory, loadModels]);

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
        }
      },
    });
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
    if (!content || !model || sending) return;
    const userMessage: ConversationMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
    };
    const history = [...messages, userMessage].map(
      ({ role, content: text }) => ({ role, content: text }),
    );
    const assistantID = crypto.randomUUID();
    setMessages((current) => [
      ...current,
      userMessage,
      { id: assistantID, role: 'assistant', content: '' },
    ]);
    setInput('');
    setSending(true);

    let conversationID = activeConversationID;
    try {
      if (!conversationID) {
        const created = await createConversation(model);
        conversationID = created.id;
        setActiveConversationID(created.id);
        setConversations((current) => [created, ...current]);
      }
      await appendConversationMessage(conversationID, {
        role: 'user',
        content,
      });
      promoteConversation(conversationID, {
        title: activeConversation?.title || generatedTitle(content),
      });
    } catch (error) {
      const detail =
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.saveMessageFailed' });
      setMessages((current) =>
        current.map((item) =>
          item.id === assistantID
            ? {
                ...item,
                content: intl.formatMessage(
                  { id: 'chat.callFailed' },
                  { detail },
                ),
              }
            : item,
        ),
      );
      setSending(false);
      return;
    }

    try {
      const result = await createChatCompletion(model, history, (streamed) =>
        setMessages((current) =>
          current.map((item) =>
            item.id === assistantID ? { ...item, content: streamed } : item,
          ),
        ),
      );
      const assistantContent =
        result.content || intl.formatMessage({ id: 'chat.emptyResponse' });
      setMessages((current) =>
        current.map((item) =>
          item.id === assistantID
            ? {
                ...item,
                content: assistantContent,
                totalTokens: result.totalTokens,
              }
            : item,
        ),
      );
      await persistAssistantMessage(
        conversationID,
        assistantContent,
        result.totalTokens,
      );
    } catch (error) {
      const detail =
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'chat.requestFailed' });
      const errorContent = intl.formatMessage(
        { id: 'chat.callFailed' },
        { detail },
      );
      setMessages((current) =>
        current.map((item) =>
          item.id === assistantID
            ? { ...item, content: errorContent }
            : item,
        ),
      );
      await persistAssistantMessage(conversationID, errorContent);
    } finally {
      setSending(false);
    }
  };

  const onComposerKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      void send();
    }
  };

  return (
    <main className={styles.chatPage}>
      <aside
        className={`${styles.historyPanel} ${historyOpen ? styles.historyPanelOpen : ''}`}
      >
        <div className={styles.historyHeader}>
          <span className={styles.historyBrand}>
            <MessageOutlined />
            {intl.formatMessage({ id: 'chat.history' })}
          </span>
          <Tooltip title={intl.formatMessage({ id: 'chat.newConversation' })}>
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
        <Button
          block
          className={styles.newConversationButton}
          disabled={sending || loadingConversation}
          icon={<PlusOutlined />}
          onClick={startNewConversation}
        >
          {intl.formatMessage({ id: 'chat.newConversation' })}
        </Button>
        <Input
          allowClear
          className={styles.conversationSearch}
          onChange={(event) => setConversationQuery(event.target.value)}
          placeholder={intl.formatMessage({ id: 'chat.searchConversation' })}
          prefix={<SearchOutlined />}
          size="small"
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
                <Tooltip title={intl.formatMessage({ id: 'common.delete' })}>
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
            {activeConversation ? (
              <Tooltip title={intl.formatMessage({ id: 'common.delete' })}>
                <Button
                  aria-label={intl.formatMessage({ id: 'common.delete' })}
                  disabled={sending || loadingConversation}
                  icon={<DeleteOutlined />}
                  onClick={() =>
                    confirmDeleteConversation(activeConversation)
                  }
                  type="text"
                />
              </Tooltip>
            ) : null}
          </div>
        </header>

        <div className={styles.messageList} ref={listRef}>
          {loadingConversation ? (
            <div className={styles.conversationLoading}>
              <Spin />
            </div>
          ) : messages.length ? (
            messages.map((item) => (
              <article
                className={`${styles.message} ${item.role === 'user' ? styles.user : ''}`}
                key={item.id}
              >
                <span className={styles.avatar}>
                  {item.role === 'user' ? <UserOutlined /> : <RobotOutlined />}
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
            ))
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
          <div className={styles.composer}>
            <Input.TextArea
              autoSize={{ minRows: 1, maxRows: 7 }}
              disabled={!model || loadingConversation}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={onComposerKeyDown}
              placeholder={intl.formatMessage({
                id: model ? 'chat.input' : 'chat.selectModelFirst',
              })}
              value={input}
            />
            <Button
              aria-label={intl.formatMessage({ id: 'chat.send' })}
              disabled={!model || !input.trim() || loadingConversation}
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
