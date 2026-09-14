import { ArrowLeftOutlined } from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { getLocale, history, useIntl, useModel, useParams } from '@umijs/max';
import { Alert, Button, Progress, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import { getAccount } from '@/services/account.api';
import type { AccountDetail } from '@/services/account.d';
import { getMyDashboard, getUserDashboard } from '@/services/dashboard.api';
import type {
  DashboardQuota,
  DashboardRecentRequest,
  MyDashboard,
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
import AvailableModelsPanel from './available-models';
import styles from './index.less';

const quotaPercent = (ratio?: number) =>
  Math.min(100, Math.round((ratio || 0) * 100));

const QuotaPanel = ({ quota }: { quota?: DashboardQuota }) => {
  const intl = useIntl();
  const tokenUnlimited = !quota?.tokenLimit;
  const requestUnlimited = !quota?.requestLimit;
  return (
    <ProCard
      className={styles.panel}
      extra={<span className={styles.panelExtra}>PERSONAL LIMITS</span>}
      title={intl.formatMessage({ id: 'dashboard.myQuota' })}
    >
      <div className={styles.quotaBlock}>
        <div className={styles.quotaHeader}>
          <span>{intl.formatMessage({ id: 'dashboard.tokenTotal' })}</span>
          <strong>
            {tokenUnlimited
              ? intl.formatMessage({ id: 'common.unlimited' })
              : `${quotaPercent(quota?.tokenUsageRatio)}%`}
          </strong>
        </div>
        <Progress
          percent={tokenUnlimited ? 0 : quotaPercent(quota?.tokenUsageRatio)}
          showInfo={false}
          strokeColor="#1677ff"
          trailColor="#f0f0f0"
        />
        <div className={styles.quotaNumber}>
          {intl.formatMessage(
            {
              id: tokenUnlimited
                ? 'dashboard.usedUnlimited'
                : 'dashboard.usedRemaining',
            },
            {
              used: formatNumber(quota?.usedTokens),
              remaining: formatNumber(quota?.remainingTokens),
            },
          )}
        </div>
      </div>
      <div className={styles.quotaBlock}>
        <div className={styles.quotaHeader}>
          <span>{intl.formatMessage({ id: 'dashboard.requestLimit' })}</span>
          <strong>
            {requestUnlimited
              ? intl.formatMessage({ id: 'common.unlimited' })
              : `${quotaPercent(quota?.requestUsageRatio)}%`}
          </strong>
        </div>
        <Progress
          percent={
            requestUnlimited ? 0 : quotaPercent(quota?.requestUsageRatio)
          }
          showInfo={false}
          strokeColor="#52c41a"
          trailColor="#f0f0f0"
        />
        <div className={styles.quotaNumber}>
          {intl.formatMessage(
            {
              id: requestUnlimited
                ? 'dashboard.usedRequestUnlimited'
                : 'dashboard.usedRemaining',
            },
            {
              used: formatNumber(quota?.usedRequests),
              remaining: formatNumber(quota?.remainingRequests),
            },
          )}
        </div>
      </div>
    </ProCard>
  );
};

const statusTag = (status: string | undefined, label: string) => {
  const colors: Record<string, string> = {
    succeeded: 'success',
    failed: 'error',
    rejected: 'warning',
    reserved: 'processing',
  };
  return <Tag color={colors[status || '']}>{label}</Tag>;
};

const MyDashboardPage = () => {
  const intl = useIntl();
  const { initialState } = useModel('@@initialState');
  const { id: accountId } = useParams<{ id?: string }>();
  const [range, setRange] = useState<DashboardRangeValue>('24h');
  const [data, setData] = useState<MyDashboard>();
  const [viewedAccount, setViewedAccount] = useState<AccountDetail>();
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setFailed(false);
    try {
      if (accountId) {
        const [dashboard, account] = await Promise.all([
          getUserDashboard({ id: accountId, range }),
          getAccount({ id: accountId }),
        ]);
        setData(dashboard);
        setViewedAccount(account);
      } else {
        setData(await getMyDashboard({ range }));
        setViewedAccount(undefined);
      }
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  }, [accountId, range]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading && !data) return <DashboardLoading />;
  if (failed && !data) return <DashboardError retry={() => void load()} />;

  const summary = data?.summary;
  const tokenStatus = data?.tokenStatus;
  const viewedUserName = accountId
    ? viewedAccount?.nickname || viewedAccount?.username || accountId
    : initialState?.currentUser?.nickname ||
      initialState?.currentUser?.username ||
      intl.formatMessage({ id: 'dashboard.hello' });
  const columns: ColumnsType<DashboardRecentRequest> = [
    {
      title: intl.formatMessage({ id: 'common.time' }),
      dataIndex: 'createdAt',
      width: 176,
      render: (value: string) => new Date(value).toLocaleString(getLocale()),
    },
    {
      title: 'Token',
      dataIndex: 'tokenPrefix',
      width: 120,
      render: (value: string) => <code>{value || '—'}</code>,
    },
    {
      title: intl.formatMessage({ id: 'common.model' }),
      dataIndex: 'modelName',
      ellipsis: true,
    },
    {
      title: intl.formatMessage({ id: 'common.status' }),
      dataIndex: 'status',
      width: 90,
      render: (status: string) =>
        statusTag(
          status,
          intl.formatMessage(
            { id: `dashboard.status.${status || 'unknown'}` },
            { status },
          ),
        ),
    },
    {
      title: intl.formatMessage({ id: 'common.tokenUsage' }),
      dataIndex: 'totalTokens',
      width: 110,
      align: 'right',
      render: formatNumber,
    },
    {
      title: intl.formatMessage({ id: 'common.duration' }),
      dataIndex: 'durationMs',
      width: 100,
      align: 'right',
      render: formatLatency,
    },
  ];
  return (
    <main className={styles.dashboardPage}>
      {accountId ? (
        <Button
          className={styles.dashboardBack}
          icon={<ArrowLeftOutlined />}
          onClick={() => history.push('/system/users')}
          type="link"
        >
          {intl.formatMessage({ id: 'system.user.backToList' })}
        </Button>
      ) : null}
      <DashboardHeader
        eyebrow="PERSONAL CONTROL PLANE"
        generatedAt={data?.generatedAt}
        onRangeChange={setRange}
        range={range}
        subtitle={intl.formatMessage({
          id: accountId
            ? 'system.user.dashboardSubtitle'
            : 'dashboard.mySubtitle',
        })}
        title={intl.formatMessage(
          {
            id: accountId ? 'system.user.dashboardTitle' : 'dashboard.myTitle',
          },
          { name: viewedUserName },
        )}
      />

      <section className={styles.metricGrid}>
        <MetricCard
          label={intl.formatMessage({ id: 'common.requests' })}
          meta={intl.formatMessage(
            { id: 'dashboard.failedCount' },
            { count: formatNumber(summary?.failedRequests) },
          )}
          value={formatNumber(summary?.totalRequests)}
        />
        <MetricCard
          label={intl.formatMessage({ id: 'common.successRate' })}
          meta={intl.formatMessage({ id: 'dashboard.upstreamOnly' })}
          tone="positive"
          value={formatPercent(summary?.successRate)}
        />
        <MetricCard
          label={intl.formatMessage({ id: 'common.tokenUsage' })}
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
          label={intl.formatMessage({ id: 'dashboard.p95Latency' })}
          meta={intl.formatMessage({ id: 'dashboard.endToEnd' })}
          value={formatLatency(summary?.p95DurationMs)}
        />
      </section>

      {data?.alerts?.length ? (
        <div className={styles.alertStack}>
          {data.alerts.map((alert) => (
            <Alert
              key={alert.code}
              message={alert.title}
              description={alert.message}
              showIcon
              type={alert.level === 'warning' ? 'warning' : 'info'}
            />
          ))}
        </div>
      ) : null}

      {!accountId ? <AvailableModelsPanel /> : null}

      <section className={styles.panelGrid}>
        <ActivityChart data={data?.trend} />
        <QuotaPanel quota={data?.quota} />
      </section>

      <section className={styles.panelGrid}>
        <ProCard
          className={styles.panel}
          extra={<span className={styles.panelExtra}>MODEL DISTRIBUTION</span>}
          title={intl.formatMessage({ id: 'dashboard.modelUsage' })}
        >
          <DistributionList data={data?.modelUsage} />
        </ProCard>
        <ProCard
          className={styles.panel}
          extra={<span className={styles.panelExtra}>API KEYS</span>}
          title={intl.formatMessage({ id: 'dashboard.tokenStatus' })}
        >
          <div className={styles.statusStrip}>
            <div className={styles.statusItem}>
              <strong>{tokenStatus?.active || 0}</strong>
              <span>{intl.formatMessage({ id: 'common.active' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong>{tokenStatus?.disabled || 0}</strong>
              <span>{intl.formatMessage({ id: 'common.disabled' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong>{tokenStatus?.expired || 0}</strong>
              <span>{intl.formatMessage({ id: 'common.expired' })}</span>
            </div>
            <div className={styles.statusItem}>
              <strong>{tokenStatus?.expiringSoon || 0}</strong>
              <span>
                {intl.formatMessage({ id: 'dashboard.expiringSoon' })}
              </span>
            </div>
          </div>
        </ProCard>
      </section>

      <ProCard
        className={styles.tablePanel}
        extra={<span className={styles.panelExtra}>LATEST 10</span>}
        title={intl.formatMessage({ id: 'dashboard.recentRequests' })}
      >
        <Table
          columns={columns}
          dataSource={data?.recentRequests || []}
          locale={{
            emptyText: intl.formatMessage({ id: 'dashboard.noRequests' }),
          }}
          pagination={false}
          rowKey="requestId"
          scroll={{ x: 850 }}
          size="small"
        />
      </ProCard>
    </main>
  );
};

export default MyDashboardPage;
