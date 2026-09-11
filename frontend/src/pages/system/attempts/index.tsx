import { SearchOutlined } from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import { Button, Input, message, Space, Table, Tag } from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import type { RouteAttemptDetail } from '@/services/control_plane.d';
import { listRouteAttempts } from '@/services/usage_management.api';
import { formatTime, SystemPage } from '../shared';

const AttemptsPage = () => {
  const intl = useIntl();
  const [data, setData] = useState<RouteAttemptDetail[]>([]);
  const [requestID, setRequestID] = useState('');
  const [loading, setLoading] = useState(true);
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
        const result = await listRouteAttempts({
          current,
          pageSize,
          requestId: requestID || undefined,
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
    [intl, pagination.pageSize, requestID],
  );

  useEffect(() => {
    void load();
  }, [load]);

  const columns: ColumnsType<RouteAttemptDetail> = [
    {
      title: intl.formatMessage({ id: 'common.time' }),
      dataIndex: 'createdAt',
      width: 170,
      render: (value: string) => formatTime(value),
    },
    {
      title: intl.formatMessage({ id: 'admin.requestId' }),
      dataIndex: 'requestId',
      width: 210,
      render: (value: string) => <code>{value}</code>,
    },
    { title: '#', dataIndex: 'sequence', width: 60 },
    {
      title: intl.formatMessage({ id: 'system.route.channel' }),
      dataIndex: 'channelName',
      render: (value: string, record) => value || record.channelId,
    },
    {
      title: 'HTTP',
      dataIndex: 'httpStatus',
      width: 100,
      render: (value: number) => (
        <Tag
          color={
            value >= 200 && value < 400
              ? 'success'
              : value
                ? 'error'
                : 'default'
          }
        >
          {value || '-'}
        </Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'common.duration' }),
      dataIndex: 'durationMs',
      width: 110,
      render: (value: number) => `${value || 0} ms`,
    },
    {
      title: intl.formatMessage({ id: 'admin.error' }),
      dataIndex: 'errorSummary',
    },
  ];

  return (
    <SystemPage pageKey="attempts">
      <Space style={{ padding: 16 }}>
        <Input
          allowClear
          onChange={(event) => setRequestID(event.target.value)}
          onPressEnter={() => void load(1)}
          placeholder={intl.formatMessage({
            id: 'system.usage.requestPlaceholder',
          })}
          style={{ width: 300 }}
          value={requestID}
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
        scroll={{ x: 1000 }}
      />
    </SystemPage>
  );
};

export default AttemptsPage;
