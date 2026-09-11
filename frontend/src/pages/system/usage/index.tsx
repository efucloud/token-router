import { SearchOutlined } from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  Button,
  Descriptions,
  Drawer,
  Input,
  message,
  Select,
  Space,
  Table,
  Tag,
} from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import type {
  RouteAttemptDetail,
  UsageLogDetail,
} from '@/services/control_plane.d';
import { listUsage, listUsageAttempts } from '@/services/usage_management.api';
import { formatTime, SystemPage } from '../shared';

const UsagePage = () => {
  const intl = useIntl();
  const [data, setData] = useState<UsageLogDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [requestID, setRequestID] = useState('');
  const [status, setStatus] = useState<string>();
  const [selected, setSelected] = useState<UsageLogDetail>();
  const [attempts, setAttempts] = useState<RouteAttemptDetail[]>([]);
  const [attemptLoading, setAttemptLoading] = useState(false);
  const [pagination, setPagination] = useState<TablePaginationConfig>({
    current: 1,
    pageSize: 20,
    total: 0,
    showSizeChanger: true,
  });

  const load = useCallback(
    async (current = 1, pageSize = pagination.pageSize || 20) => {
      setLoading(true);
      try {
        const result = await listUsage({
          current,
          pageSize,
          requestId: requestID || undefined,
          status,
        });
        setData(result.data || []);
        setPagination((value) => ({
          ...value,
          current,
          pageSize,
          total: result.total || 0,
        }));
      } catch {
        message.error(intl.formatMessage({ id: 'system.loadFailed' }));
      } finally {
        setLoading(false);
      }
    },
    [intl, pagination.pageSize, requestID, status],
  );

  useEffect(() => {
    void load();
  }, [load]);

  const showDetail = async (record: UsageLogDetail) => {
    setSelected(record);
    setAttemptLoading(true);
    try {
      const result = await listUsageAttempts({
        requestId: record.requestId || '',
      });
      setAttempts(result.data || []);
    } catch {
      setAttempts([]);
    } finally {
      setAttemptLoading(false);
    }
  };

  const columns: ColumnsType<UsageLogDetail> = [
    {
      title: intl.formatMessage({ id: 'common.time' }),
      dataIndex: 'createdAt',
      width: 170,
      render: (value: string) => formatTime(value),
    },
    {
      title: intl.formatMessage({ id: 'admin.requestId' }),
      dataIndex: 'requestId',
      width: 190,
      render: (value: string, record) => (
        <Button
          onClick={() => void showDetail(record)}
          size="small"
          type="link"
        >
          <code>{value}</code>
        </Button>
      ),
    },
    {
      title: intl.formatMessage({ id: 'admin.user' }),
      width: 150,
      render: (_, record) => record.username || record.accountId,
    },
    {
      title: intl.formatMessage({ id: 'common.model' }),
      dataIndex: 'modelName',
      width: 180,
    },
    {
      title: intl.formatMessage({ id: 'system.usage.channel' }),
      dataIndex: 'channelName',
      width: 150,
      render: (value: string, record) => value || record.channelId || '-',
    },
    {
      title: intl.formatMessage({ id: 'common.status' }),
      dataIndex: 'status',
      width: 110,
      render: (value: string, record) => (
        <Tag
          color={
            value === 'succeeded'
              ? 'success'
              : value === 'failed'
                ? 'error'
                : 'default'
          }
        >
          {value} {record.httpStatus ? `· ${record.httpStatus}` : ''}
        </Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'common.tokenUsage' }),
      width: 170,
      render: (_, record) =>
        `${record.totalTokens || 0} (${record.promptTokens || 0} / ${record.completionTokens || 0})`,
    },
    {
      title: intl.formatMessage({ id: 'common.duration' }),
      dataIndex: 'durationMs',
      width: 100,
      render: (value: number) => `${value || 0} ms`,
    },
  ];

  const attemptColumns: ColumnsType<RouteAttemptDetail> = [
    { title: '#', dataIndex: 'sequence', width: 60 },
    {
      title: intl.formatMessage({ id: 'system.usage.channel' }),
      dataIndex: 'channelName',
    },
    { title: 'HTTP', dataIndex: 'httpStatus', width: 80 },
    {
      title: intl.formatMessage({ id: 'common.duration' }),
      dataIndex: 'durationMs',
      width: 100,
      render: (value: number) => `${value || 0} ms`,
    },
    {
      title: intl.formatMessage({ id: 'admin.error' }),
      dataIndex: 'errorSummary',
    },
  ];

  return (
    <SystemPage pageKey="usage">
      <Space style={{ padding: 16 }} wrap>
        <Input
          allowClear
          onChange={(event) => setRequestID(event.target.value)}
          onPressEnter={() => void load(1)}
          placeholder={intl.formatMessage({
            id: 'system.usage.requestPlaceholder',
          })}
          style={{ width: 280 }}
          value={requestID}
        />
        <Select
          allowClear
          onChange={setStatus}
          options={['succeeded', 'failed', 'rejected', 'reserved'].map(
            (value) => ({
              label: value,
              value,
            }),
          )}
          placeholder={intl.formatMessage({ id: 'common.status' })}
          style={{ width: 150 }}
          value={status}
        />
        <Button
          icon={<SearchOutlined />}
          onClick={() => void load(1)}
          type="primary"
        >
          {intl.formatMessage({ id: 'common.search' })}
        </Button>
      </Space>
      <Table
        columns={columns}
        dataSource={data}
        loading={loading}
        onChange={(page) => void load(page.current, page.pageSize)}
        pagination={pagination}
        rowKey="id"
        scroll={{ x: 1220 }}
      />
      <Drawer
        onClose={() => setSelected(undefined)}
        open={Boolean(selected)}
        title={intl.formatMessage({ id: 'system.usage.detail' })}
        width={760}
      >
        <Descriptions bordered column={2} size="small">
          <Descriptions.Item
            label={intl.formatMessage({ id: 'admin.requestId' })}
            span={2}
          >
            <code>{selected?.requestId}</code>
          </Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'admin.user' })}>
            {selected?.username || selected?.accountId}
          </Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'common.model' })}>
            {selected?.modelName}
          </Descriptions.Item>
          <Descriptions.Item
            label={intl.formatMessage({ id: 'common.status' })}
          >
            {selected?.status} / {selected?.httpStatus || '-'}
          </Descriptions.Item>
          <Descriptions.Item
            label={intl.formatMessage({ id: 'common.duration' })}
          >
            {selected?.durationMs || 0} ms
          </Descriptions.Item>
          <Descriptions.Item
            label={intl.formatMessage({ id: 'common.tokenUsage' })}
          >
            {selected?.totalTokens || 0}
          </Descriptions.Item>
          <Descriptions.Item
            label={intl.formatMessage({ id: 'system.usage.endpoint' })}
          >
            {selected?.endpoint}
          </Descriptions.Item>
          {selected?.errorSummary ? (
            <Descriptions.Item
              label={intl.formatMessage({ id: 'admin.error' })}
              span={2}
            >
              {selected.errorCode}: {selected.errorSummary}
            </Descriptions.Item>
          ) : null}
        </Descriptions>
        <h3 style={{ marginTop: 24 }}>
          {intl.formatMessage({ id: 'system.attempts.title' })}
        </h3>
        <Table
          columns={attemptColumns}
          dataSource={attempts}
          loading={attemptLoading}
          pagination={false}
          rowKey="id"
          size="small"
        />
      </Drawer>
    </SystemPage>
  );
};

export default UsagePage;
