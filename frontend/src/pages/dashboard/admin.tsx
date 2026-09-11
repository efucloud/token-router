import { ProCard } from '@ant-design/pro-components';
import { getLocale, useIntl, useModel } from '@umijs/max';
import { Badge, Result, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import { getAdminDashboard } from '@/services/dashboard.api';
import type {
  AdminDashboard,
  DashboardDimensionUsage,
  DashboardRecentRequest,
} from '@/services/dashboard.d';
import {
  ActivityChart,
  DashboardError,
  DashboardHeader,
  DashboardLoading,
  DistributionList,
  formatLatency,
  formatNumber,
  formatPercent,
  MetricCard,
  type DashboardRangeValue,
} from './components';
import styles from './index.less';

const AdminDashboardPage = () => {
  const intl = useIntl();
  const { initialState } = useModel('@@initialState');
  const [range, setRange] = useState<DashboardRangeValue>('24h');
  const [data, setData] = useState<AdminDashboard>();
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setFailed(false);
    try {
      setData(await getAdminDashboard({ range }));
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  }, [range]);

  useEffect(() => {
    if (initialState?.currentUser?.role === 'admin') void load();
  }, [initialState?.currentUser?.role, load]);

  if (initialState?.currentUser?.role !== 'admin') {
    return (
      <Result
        status="403"
        subTitle={intl.formatMessage({ id: 'admin.forbiddenHint' })}
        title={intl.formatMessage({ id: 'admin.forbiddenTitle' })}
      />
    );
  }
  if (loading && !data) return <DashboardLoading />;
  if (failed && !data) return <DashboardError retry={() => void load()} />;

  const summary = data?.summary;
  const health = data?.resourceHealth;
  const principals = data?.principals;
  const routing = data?.routing;
  const usageColumns: ColumnsType<DashboardDimensionUsage> = [
    {
      title: intl.formatMessage({ id: 'admin.user' }),
      dataIndex: 'name',
      ellipsis: true,
    },
    {
      title: intl.formatMessage({ id: 'common.requests' }),
      dataIndex: 'requests',
      width: 100,
      align: 'right',
      render: formatNumber,
    },
    {
      title: 'Token',
      dataIndex: 'totalTokens',
      width: 110,
      align: 'right',
      render: formatNumber,
    },
    {
      title: intl.formatMessage({ id: 'common.successRate' }),
      dataIndex: 'successRate',
      width: 100,
      align: 'right',
      render: formatPercent,
    },
    {
      title: intl.formatMessage({ id: 'admin.quotaUsage' }),
      dataIndex: 'usageRatio',
      width: 100,
      align: 'right',
      render: formatPercent,
    },
  ];
  const failureColumns: ColumnsType<DashboardRecentRequest> = [
    {
      title: intl.formatMessage({ id: 'common.time' }),
      dataIndex: 'createdAt',
      width: 176,
      render: (value: string) => new Date(value).toLocaleString(getLocale()),
    },
    {
      title: intl.formatMessage({ id: 'common.model' }),
      dataIndex: 'modelName',
      ellipsis: true,
    },
    {
      title: intl.formatMessage({ id: 'admin.error' }),
      dataIndex: 'errorCode',
      width: 170,
      render: (value: string) => (
        <Tag color="error">{value || 'upstream_error'}</Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'admin.attempts' }),
      dataIndex: 'attemptCount',
      width: 80,
      align: 'right',
    },
    {
      title: intl.formatMessage({ id: 'common.duration' }),
      dataIndex: 'durationMs',
      width: 100,
      align: 'right',
      render: formatLatency,
    },
    {
      title: intl.formatMessage({ id: 'admin.requestId' }),
      dataIndex: 'requestId',
      width: 190,
      ellipsis: true,
      render: (value: string) => <code>{value}</code>,
    },
  ];
  return (
    <main className={styles.dashboardPage}>
      <DashboardHeader
        eyebrow="ENTERPRISE GATEWAY OPERATIONS"
        generatedAt={data?.generatedAt}
        onRangeChange={setRange}
        range={range}
        subtitle={intl.formatMessage({ id: 'admin.subtitle' })}
        title={intl.formatMessage({ id: 'admin.title' })}
      />

      <section className={styles.metricGrid}>
        <MetricCard
          label={intl.formatMessage({ id: 'admin.globalRequests' })}
          meta={intl.formatMessage(
            { id: 'admin.failedRejected' },
            {
              failed: formatNumber(summary?.failedRequests),
              rejected: formatNumber(summary?.rejectedRequests),
            },
          )}
          value={formatNumber(summary?.totalRequests)}
        />
        <MetricCard
          label={intl.formatMessage({ id: 'common.successRate' })}
          meta={`P95 ${formatLatency(summary?.p95DurationMs)}`}
          tone="positive"
          value={formatPercent(summary?.successRate)}
        />
        <MetricCard
          label={intl.formatMessage({ id: 'admin.tokenThroughput' })}
          meta={intl.formatMessage(
            { id: 'dashboard.inputOutput' },
            {
              input: formatNumber(summary?.promptTokens),
              output: formatNumber(summary?.completionTokens),
            },
          )}
          value={formatNumber(summary?.totalTokens)}
        />
        <MetricCard
          label={intl.formatMessage({ id: 'admin.activePrincipals' })}
          meta={intl.formatMessage(
            { id: 'admin.activeUsers' },
            {
              tokens: formatNumber(principals?.activeTokens),
              users: formatNumber(principals?.activeUsers),
            },
          )}
          value={formatNumber(principals?.activeUsers)}
        />
      </section>

      <section className={styles.panelGrid}>
        <ActivityChart data={data?.trend} />
        <ProCard
          className={styles.panel}
          extra={<span className={styles.panelExtra}>RESOURCE HEALTH</span>}
          title={intl.formatMessage({ id: 'admin.resourceHealth' })}
        >
          <div className={styles.statusStrip}>
            <div className={styles.statusItem}>
              <strong>{health?.activeModels || 0}</strong>
              <span>{intl.formatMessage({ id: 'admin.activeModels' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong className={styles.positive}>
                {health?.healthyChannels || 0}
              </strong>
              <span>{intl.formatMessage({ id: 'admin.healthyChannels' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong>{health?.unknownChannels || 0}</strong>
              <span>{intl.formatMessage({ id: 'admin.unknownChannels' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong className={styles.warning}>
                {health?.cooldownChannels || 0}
              </strong>
              <span>
                {intl.formatMessage({ id: 'admin.cooldownChannels' })}
              </span>
            </div>
            <div className={styles.statusItem}>
              <strong>{health?.disabledChannels || 0}</strong>
              <span>{intl.formatMessage({ id: 'common.disabled' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong className={styles.warning}>
                {health?.misconfiguredChannels || 0}
              </strong>
              <span>
                {intl.formatMessage({ id: 'admin.misconfiguredChannels' })}
              </span>
            </div>
          </div>
        </ProCard>
      </section>

      <section className={styles.panelGrid}>
        <ProCard
          className={styles.panel}
          extra={<span className={styles.panelExtra}>MODEL DISTRIBUTION</span>}
          title={intl.formatMessage({ id: 'admin.modelDistribution' })}
        >
          <DistributionList data={data?.modelUsage} />
        </ProCard>
        <ProCard
          className={styles.panel}
          extra={<span className={styles.panelExtra}>ROUTING</span>}
          title={intl.formatMessage({ id: 'admin.routingRisk' })}
        >
          <div className={styles.statusStrip}>
            <div className={styles.statusItem}>
              <strong>{formatNumber(routing?.upstreamAttempts)}</strong>
              <span>
                {intl.formatMessage({ id: 'admin.upstreamAttempts' })}
              </span>
            </div>
            <div className={styles.statusItem}>
              <strong>{formatNumber(routing?.failoverRequests)}</strong>
              <span>{intl.formatMessage({ id: 'admin.failovers' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong>{formatPercent(routing?.failoverSuccessRate)}</strong>
              <span>{intl.formatMessage({ id: 'admin.failoverRate' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong>{principals?.usersNearQuota || 0}</strong>
              <span>{intl.formatMessage({ id: 'admin.userQuotaRisk' })}</span>
            </div>
          </div>
          <div style={{ marginTop: 18 }}>
            <Badge
              color="#faad14"
              text={intl.formatMessage(
                { id: 'admin.tokensNearQuota' },
                { count: principals?.tokensNearQuota || 0 },
              )}
            />
            <br />
            <Badge
              color="#8c8c8c"
              text={intl.formatMessage(
                { id: 'admin.tokensExpiring' },
                { count: principals?.tokensExpiringSoon || 0 },
              )}
            />
          </div>
        </ProCard>
      </section>

      <section className={styles.panelGrid}>
        <ProCard
          className={styles.tablePanel}
          extra={<span className={styles.panelExtra}>TOP 10 BY USAGE</span>}
          title={intl.formatMessage({ id: 'admin.userUsage' })}
        >
          <Table
            columns={usageColumns}
            dataSource={data?.userUsage || []}
            locale={{
              emptyText: intl.formatMessage({ id: 'admin.noUserUsage' }),
            }}
            pagination={false}
            rowKey="id"
            size="small"
          />
        </ProCard>
        <ProCard
          className={styles.panel}
          extra={<span className={styles.panelExtra}>CHANNEL ATTEMPTS</span>}
          title={intl.formatMessage({ id: 'admin.channelAttempts' })}
        >
          <DistributionList data={data?.channelUsage} valueLabel="requests" />
        </ProCard>
      </section>

      <ProCard
        className={styles.tablePanel}
        extra={<span className={styles.panelExtra}>LATEST FAILURES</span>}
        title={intl.formatMessage({ id: 'admin.recentFailures' })}
      >
        <Table
          columns={failureColumns}
          dataSource={data?.recentFailures || []}
          locale={{ emptyText: intl.formatMessage({ id: 'admin.noFailures' }) }}
          pagination={false}
          rowKey="requestId"
          scroll={{ x: 920 }}
          size="small"
        />
      </ProCard>
    </main>
  );
};

export default AdminDashboardPage;
