import {
  ApiOutlined,
  ArrowRightOutlined,
  ReloadOutlined,
  RobotOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { history, useIntl } from '@umijs/max';
import { Button, Empty, Input, Skeleton, Tag } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { type GatewayModel, listGatewayModels } from '@/data-plane/client';
import styles from './index.less';

const formatTokenLimit = (value?: number) => {
  if (!value) return '—';
  if (value >= 1_000_000) return `${value / 1_000_000}M`;
  if (value >= 1_000) return `${value / 1_000}K`;
  return value.toLocaleString();
};

const AvailableModelsPanel = () => {
  const intl = useIntl();
  const [models, setModels] = useState<GatewayModel[]>([]);
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setFailed(false);
    try {
      setModels(await listGatewayModels());
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const filteredModels = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return models;
    return models.filter((model) =>
      [
        model.id,
        model.display_name,
        model.description,
        model.modality,
        ...(model.capabilities || []),
      ]
        .filter(Boolean)
        .some((value) => value?.toLowerCase().includes(normalized)),
    );
  }, [models, query]);

  const capabilityLabel = (capability: string) =>
    intl.formatMessage({
      id: `dashboard.modelCapability.${capability}`,
      defaultMessage: capability,
    });

  return (
    <ProCard className={`${styles.panel} ${styles.availableModelsPanel}`}>
      <div className={styles.modelPanelHeader}>
        <div>
          <div className={styles.modelPanelTitleRow}>
            <span className={styles.modelPulse} />
            <h2>{intl.formatMessage({ id: 'dashboard.availableModels' })}</h2>
            {!loading && !failed ? (
              <span className={styles.modelCount}>
                {intl.formatMessage(
                  { id: 'dashboard.availableModelCount' },
                  { count: models.length },
                )}
              </span>
            ) : null}
          </div>
          <p>{intl.formatMessage({ id: 'dashboard.availableModelsHint' })}</p>
        </div>
        <div className={styles.modelPanelActions}>
          <Button
            aria-label={intl.formatMessage({ id: 'common.refresh' })}
            icon={<ReloadOutlined />}
            loading={loading}
            onClick={() => void load()}
          />
          <Button
            icon={<ArrowRightOutlined />}
            iconPosition="end"
            onClick={() => history.push('/personal/chat')}
            type="primary"
          >
            {intl.formatMessage({ id: 'dashboard.startChat' })}
          </Button>
        </div>
      </div>

      {!loading && !failed && models.length > 4 ? (
        <Input
          allowClear
          aria-label={intl.formatMessage({ id: 'dashboard.searchModels' })}
          className={styles.modelSearch}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={intl.formatMessage({ id: 'dashboard.searchModels' })}
          prefix={<SearchOutlined />}
          value={query}
        />
      ) : null}

      {loading ? (
        <div className={styles.modelCardGrid}>
          {[0, 1, 2].map((item) => (
            <div className={styles.modelCardSkeleton} key={item}>
              <Skeleton
                active
                paragraph={{ rows: 2 }}
                title={{ width: '54%' }}
              />
            </div>
          ))}
        </div>
      ) : failed ? (
        <div className={styles.modelPanelState}>
          <Empty
            description={intl.formatMessage({
              id: 'dashboard.availableModelsFailed',
            })}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              {intl.formatMessage({ id: 'dashboard.reload' })}
            </Button>
          </Empty>
        </div>
      ) : filteredModels.length ? (
        <div className={styles.modelCardGrid}>
          {filteredModels.map((model) => (
            <article className={styles.modelCard} key={model.id}>
              <div className={styles.modelCardTop}>
                <span className={styles.modelGlyph}>
                  {model.modality === 'embedding' ? (
                    <ApiOutlined />
                  ) : (
                    <RobotOutlined />
                  )}
                </span>
                <div className={styles.modelIdentity}>
                  <strong>{model.display_name || model.id}</strong>
                  <code>{model.id}</code>
                </div>
                <Tag bordered={false} color="success">
                  {intl.formatMessage({ id: 'dashboard.modelAvailable' })}
                </Tag>
              </div>
              <p className={styles.modelDescription}>
                {model.description ||
                  intl.formatMessage({ id: 'dashboard.modelDefaultHint' })}
              </p>
              <div className={styles.modelMetadata}>
                <span>
                  <small>
                    {intl.formatMessage({ id: 'system.model.modality' })}
                  </small>
                  <b>{model.modality || '—'}</b>
                </span>
                <span>
                  <small>
                    {intl.formatMessage({ id: 'system.model.contextWindow' })}
                  </small>
                  <b>{formatTokenLimit(model.context_window)}</b>
                </span>
                <span>
                  <small>
                    {intl.formatMessage({ id: 'system.model.maxOutputTokens' })}
                  </small>
                  <b>{formatTokenLimit(model.max_output_tokens)}</b>
                </span>
              </div>
              {model.capabilities?.length ? (
                <div className={styles.modelCapabilities}>
                  {model.capabilities.map((capability) => (
                    <Tag key={capability}>{capabilityLabel(capability)}</Tag>
                  ))}
                </div>
              ) : null}
            </article>
          ))}
        </div>
      ) : (
        <div className={styles.modelPanelState}>
          <Empty
            description={intl.formatMessage({
              id: query
                ? 'dashboard.noMatchingModels'
                : 'dashboard.noAvailableModels',
            })}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </div>
      )}
    </ProCard>
  );
};

export default AvailableModelsPanel;
