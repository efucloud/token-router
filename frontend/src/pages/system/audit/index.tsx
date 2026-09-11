import { SearchOutlined } from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  Button,
  Input,
  message,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import { listAuditLogs } from '@/services/audit_management.api';
import type { AuditLogDetail } from '@/services/control_plane.d';
import { formatTime, SystemPage } from '../shared';

const AuditPage = () => {
  const intl = useIntl();
  const [data, setData] = useState<AuditLogDetail[]>([]);
  const [operatorID, setOperatorID] = useState('');
  const [resourceType, setResourceType] = useState<string>();
  const [resultFilter, setResultFilter] = useState<string>();
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
        const result = await listAuditLogs({
          current,
          operatorId: operatorID || undefined,
          pageSize,
          resourceType,
          result: resultFilter,
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
    [intl, operatorID, pagination.pageSize, resourceType, resultFilter],
  );

  useEffect(() => {
    void load();
  }, [load]);

  const columns: ColumnsType<AuditLogDetail> = [
    {
      title: intl.formatMessage({ id: 'common.time' }),
      dataIndex: 'createdAt',
      width: 170,
      render: (value: string) => formatTime(value),
    },
    {
      title: intl.formatMessage({ id: 'system.audit.operator' }),
      width: 180,
      render: (_, record) => (
        <span>
          <strong>{record.operatorUsername || '-'}</strong>
          <br />
          <Typography.Text type="secondary">
            {record.operatorId}
          </Typography.Text>
        </span>
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.audit.action' }),
      dataIndex: 'action',
      width: 120,
      render: (value: string) => <Tag>{value}</Tag>,
    },
    {
      title: intl.formatMessage({ id: 'system.audit.resource' }),
      width: 190,
      render: (_, record) => (
        <span>
          {record.resourceType || '-'}
          {record.resourceId ? (
            <>
              <br />
              <code>{record.resourceId}</code>
            </>
          ) : null}
        </span>
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.audit.result' }),
      width: 120,
      render: (_, record) => (
        <Tag color={record.result === 'succeeded' ? 'success' : 'error'}>
          {record.result === 'succeeded'
            ? intl.formatMessage({ id: 'system.audit.succeeded' })
            : intl.formatMessage({ id: 'system.audit.failed' })}
          {record.statusCode ? ` · ${record.statusCode}` : ''}
        </Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'admin.requestId' }),
      dataIndex: 'requestId',
      width: 210,
      render: (value: string) => (
        <Typography.Text copyable>{value}</Typography.Text>
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.audit.source' }),
      dataIndex: 'remoteIp',
      width: 140,
    },
  ];

  return (
    <SystemPage pageKey="audit">
      <Space style={{ padding: 16 }} wrap>
        <Input
          allowClear
          onChange={(event) => setOperatorID(event.target.value)}
          onPressEnter={() => void load(1)}
          placeholder={intl.formatMessage({
            id: 'system.audit.operatorPlaceholder',
          })}
          style={{ width: 240 }}
          value={operatorID}
        />
        <Select
          allowClear
          onChange={setResourceType}
          options={['models', 'providers', 'channels', 'routes', 'account'].map(
            (value) => ({
              label: value,
              value,
            }),
          )}
          placeholder={intl.formatMessage({ id: 'system.audit.resource' })}
          style={{ width: 150 }}
          value={resourceType}
        />
        <Select
          allowClear
          onChange={setResultFilter}
          options={[
            {
              label: intl.formatMessage({ id: 'system.audit.succeeded' }),
              value: 'succeeded',
            },
            {
              label: intl.formatMessage({ id: 'system.audit.failed' }),
              value: 'failed',
            },
          ]}
          placeholder={intl.formatMessage({ id: 'system.audit.result' })}
          style={{ width: 130 }}
          value={resultFilter}
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
        scroll={{ x: 1200 }}
      />
    </SystemPage>
  );
};

export default AuditPage;
