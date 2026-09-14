import {
  ApiOutlined,
  ArrowRightOutlined,
  ReloadOutlined,
  RobotOutlined,
  SearchOutlined,
} from '@ant-design/icons';
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

const AvailableModelsPage = () => {
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
        ...(model.capabilities || []).flatMap((capability) => [
          capability,
          intl.formatMessage({
            id: `models.capability.${capability}`,
            defaultMessage: capability,
          }),
        ]),
      ]
        .filter(Boolean)
        .some((value) => value?.toLowerCase().includes(normalized)),
    );
  }, [intl, models, query]);

  const capabilityLabel = (capability: string) =>
    intl.formatMessage({
      id: `models.capability.${capability}`,
      defaultMessage: capability,
    });

  return (
    <main className={styles.modelsPage}>
      <header className={styles.catalogHeader}>
        <div className={styles.catalogCopy}>
          <span className={styles.eyebrow}>MODEL CATALOG</span>
          <h1>{intl.formatMessage({ id: 'models.title' })}</h1>
          <p>{intl.formatMessage({ id: 'models.description' })}</p>
          {!loading && !failed ? (
            <span className={styles.availabilityBadge}>
              <i />
              {intl.formatMessage(
                { id: 'models.availableCount' },
                { count: models.length },
              )}
            </span>
          ) : null}
        </div>
        <div className={styles.catalogActions}>
          <Button
            aria-label={intl.formatMessage({ id: 'common.refresh' })}
            icon={<ReloadOutlined />}
            loading={loading}
            onClick={() => void load()}
          >
            {intl.formatMessage({ id: 'common.refresh' })}
          </Button>
          <Button
            icon={<ArrowRightOutlined />}
            iconPosition="end"
            onClick={() => history.push('/personal/chat')}
            type="primary"
          >
            {intl.formatMessage({ id: 'models.startChat' })}
          </Button>
        </div>
      </header>

      <section className={styles.catalogBody}>
        <div className={styles.catalogToolbar}>
          <div>
            <strong>{intl.formatMessage({ id: 'models.catalog' })}</strong>
            <span>{intl.formatMessage({ id: 'models.catalogHint' })}</span>
          </div>
          <Input
            allowClear
            aria-label={intl.formatMessage({ id: 'models.search' })}
            className={styles.modelSearch}
            disabled={loading || failed || models.length === 0}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={intl.formatMessage({ id: 'models.search' })}
            prefix={<SearchOutlined />}
            value={query}
          />
        </div>

        {loading ? (
          <div className={styles.modelCardGrid}>
            {[0, 1, 2, 3, 4, 5].map((item) => (
              <div className={styles.modelCardSkeleton} key={item}>
                <Skeleton
                  active
                  paragraph={{ rows: 3 }}
                  title={{ width: '54%' }}
                />
              </div>
            ))}
          </div>
        ) : failed ? (
          <div className={styles.catalogState}>
            <Empty
              description={intl.formatMessage({ id: 'models.loadFailed' })}
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
                    {intl.formatMessage({ id: 'models.available' })}
                  </Tag>
                </div>
                <p className={styles.modelDescription}>
                  {model.description ||
                    intl.formatMessage({ id: 'models.defaultHint' })}
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
                      {intl.formatMessage({
                        id: 'system.model.maxOutputTokens',
                      })}
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
          <div className={styles.catalogState}>
            <Empty
              description={intl.formatMessage({
                id: query ? 'models.noMatching' : 'models.empty',
              })}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          </div>
        )}
      </section>
    </main>
  );
};

export default AvailableModelsPage;
