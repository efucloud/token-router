import { Line } from '@ant-design/plots';
import { ReloadOutlined } from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { getLocale, useIntl } from '@umijs/max';
import { Button, Empty, Result, Segmented, Skeleton, Typography } from 'antd';
import type { ReactNode } from 'react';
import type {
  DashboardDimensionUsage,
  DashboardTrendPoint,
} from '@/services/dashboard.d';
import styles from './index.less';

export type DashboardRangeValue = '24h' | '7d' | '30d';

export const formatNumber = (value?: number) =>
  new Intl.NumberFormat(getLocale(), {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value || 0);
export const formatPercent = (value?: number) =>
  value == null ? '—' : `${(value * 100).toFixed(1)}%`;
export const formatLatency = (value?: number) =>
  value == null ? '—' : `${Math.round(value)} ms`;

export const MetricCard = ({
  label,
  value,
  meta,
  tone,
}: {
  label: string;
  value: ReactNode;
  meta: string;
  tone?: 'positive' | 'warning';
}) => (
  <div className={styles.metricCard}>
    <div className={styles.metricLabel}>{label}</div>
    <div className={`${styles.metricValue} ${tone ? styles[tone] : ''}`}>
      {value}
    </div>
    <span className={styles.metricMeta}>{meta}</span>
  </div>
);

export const DashboardHeader = ({
  eyebrow,
  title,
  subtitle,
  range,
  generatedAt,
  onRangeChange,
}: {
  eyebrow: string;
  title: string;
  subtitle: string;
  range: DashboardRangeValue;
  generatedAt?: string;
  onRangeChange: (value: DashboardRangeValue) => void;
}) => {
  const intl = useIntl();
  return (
    <div className={styles.pageHeader}>
      <div>
        <span className={styles.eyebrow}>{eyebrow}</span>
        <Typography.Title className={styles.title} level={1}>
          {title}
        </Typography.Title>
        <span className={styles.subtitle}>{subtitle}</span>
        {generatedAt ? (
          <span className={styles.updatedAt}>
            {intl.formatMessage(
              { id: 'dashboard.generatedAt' },
              { time: new Date(generatedAt).toLocaleString(getLocale()) },
            )}
          </span>
        ) : null}
      </div>
      <Segmented<DashboardRangeValue>
        onChange={onRangeChange}
        options={[
          {
            label: intl.formatMessage({ id: 'dashboard.range.24h' }),
            value: '24h',
          },
          {
            label: intl.formatMessage({ id: 'dashboard.range.7d' }),
            value: '7d',
          },
          {
            label: intl.formatMessage({ id: 'dashboard.range.30d' }),
            value: '30d',
          },
        ]}
        value={range}
      />
    </div>
  );
};

export const DashboardLoading = () => (
  <div className={styles.dashboardPage}>
    <Skeleton active paragraph={{ rows: 12 }} title={{ width: 280 }} />
  </div>
);

export const DashboardError = ({ retry }: { retry: () => void }) => {
  const intl = useIntl();
  return (
    <div className={styles.dashboardPage}>
      <Result
        extra={
          <Button icon={<ReloadOutlined />} onClick={retry}>
            {intl.formatMessage({ id: 'dashboard.reload' })}
          </Button>
        }
        status="error"
        subTitle={intl.formatMessage({ id: 'dashboard.loadFailedHint' })}
        title={intl.formatMessage({ id: 'dashboard.loadFailed' })}
      />
    </div>
  );
};

export const ActivityChart = ({ data }: { data?: DashboardTrendPoint[] }) => {
  const intl = useIntl();
  const chartData = (data || []).flatMap((point) => [
    {
      bucket: point.bucket || '',
      type: intl.formatMessage({ id: 'dashboard.allRequests' }),
      value: point.requests || 0,
    },
    {
      bucket: point.bucket || '',
      type: intl.formatMessage({ id: 'dashboard.succeededRequests' }),
      value: point.succeeded || 0,
    },
  ]);
  return (
    <ProCard
      className={styles.panel}
      extra={<span className={styles.panelExtra}>REQUEST VOLUME</span>}
      title={intl.formatMessage({ id: 'dashboard.activity' })}
    >
      {chartData.length ? (
        <Line
          axis={{
            x: {
              labelFormatter: (value: string) =>
                new Date(value).toLocaleDateString(undefined, {
                  month: '2-digit',
                  day: '2-digit',
                  hour: '2-digit',
                }),
            },
            y: { title: false },
          }}
          colorField="type"
          data={chartData}
          height={260}
          legend={{ color: { position: 'top' } }}
          scale={{ color: { range: ['#1677ff', '#52c41a'] } }}
          style={{ lineWidth: 2 }}
          xField="bucket"
          yField="value"
        />
      ) : (
        <div className={styles.emptyState}>
          <Empty
            description={intl.formatMessage({ id: 'dashboard.noActivity' })}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </div>
      )}
    </ProCard>
  );
};

export const DistributionList = ({
  data,
  valueLabel = 'tokens',
}: {
  data?: DashboardDimensionUsage[];
  valueLabel?: 'tokens' | 'requests';
}) => {
  const intl = useIntl();
  const values = data || [];
  const max = Math.max(
    1,
    ...values.map((item) =>
      valueLabel === 'tokens' ? item.totalTokens || 0 : item.requests || 0,
    ),
  );
  if (!values.length) {
    return (
      <Empty
        description={intl.formatMessage({ id: 'dashboard.noDistribution' })}
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      />
    );
  }
  return (
    <div className={styles.distributionList}>
      {values.map((item) => {
        const value =
          valueLabel === 'tokens' ? item.totalTokens || 0 : item.requests || 0;
        return (
          <div className={styles.distributionItem} key={item.id}>
            <span className={styles.distributionName}>
              {item.name || item.id}
            </span>
            <span className={styles.distributionBar}>
              <span style={{ width: `${Math.max(3, (value / max) * 100)}%` }} />
            </span>
            <span className={styles.distributionValue}>
              {formatNumber(value)}
            </span>
          </div>
        );
      })}
    </div>
  );
};
