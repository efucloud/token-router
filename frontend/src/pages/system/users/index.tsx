import {
  DashboardOutlined,
  EditOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { history, useIntl } from '@umijs/max';
import {
  Button,
  Form,
  InputNumber,
  message,
  Modal,
  Progress,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import {
  changeAccountStatus,
  listAccount,
  setAccountRole,
  updateAccountQuota,
} from '@/services/account.api';
import type { AccountDetail, AccountQuotaUpdate } from '@/services/account.d';
import { SystemPage } from '../shared';

const percent = (used?: number, limit?: number) =>
  limit ? Math.min(100, Math.round(((used || 0) / limit) * 100)) : 0;

const UsersPage = () => {
  const intl = useIntl();
  const [data, setData] = useState<AccountDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState<AccountDetail>();
  const [pagination, setPagination] = useState<TablePaginationConfig>({
    current: 1,
    pageSize: 20,
    total: 0,
    showSizeChanger: true,
  });
  const [form] = Form.useForm<AccountQuotaUpdate>();

  const load = useCallback(
    async (
      current = pagination.current || 1,
      pageSize = pagination.pageSize || 20,
    ) => {
      setLoading(true);
      try {
        const result = await listAccount({
          current,
          pageSize,
          order: 'created_at desc',
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
    [intl, pagination.current, pagination.pageSize],
  );

  useEffect(() => {
    void load();
  }, [load]);

  const changeRole = async (record: AccountDetail, role: string) => {
    try {
      await setAccountRole({ ids: [record.id], role });
      await load();
      message.success(intl.formatMessage({ id: 'system.saved' }));
    } catch {
      message.error(intl.formatMessage({ id: 'system.saveFailed' }));
    }
  };

  const toggle = async (record: AccountDetail) => {
    try {
      await changeAccountStatus({ ids: [record.id], enable: !record.enable });
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'system.saveFailed' }));
    }
  };

  const showQuota = (record: AccountDetail) => {
    setEditing(record);
    form.setFieldsValue({
      tokenLimit: record.tokenLimit || 0,
      requestLimit: record.requestLimit || 0,
    });
  };

  const saveQuota = async () => {
    if (!editing) return;
    const values = await form.validateFields();
    setSaving(true);
    try {
      await updateAccountQuota({ id: editing.id }, values);
      setEditing(undefined);
      await load();
      message.success(intl.formatMessage({ id: 'system.saved' }));
    } catch {
      message.error(intl.formatMessage({ id: 'system.saveFailed' }));
    } finally {
      setSaving(false);
    }
  };

  const columns: ColumnsType<AccountDetail> = [
    {
      title: intl.formatMessage({ id: 'system.user.identity' }),
      render: (_, record) => (
        <span>
          <strong>{record.nickname || record.username}</strong>
          <br />
          <Typography.Text type="secondary">
            {record.username} · {record.email || '-'}
          </Typography.Text>
        </span>
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.user.role' }),
      dataIndex: 'role',
      width: 150,
      render: (value: string, record) => (
        <Select
          onChange={(role) => void changeRole(record, role)}
          options={['admin', 'edit', 'view', 'none'].map((role) => ({
            label: intl.formatMessage({ id: `system.role.${role}` }),
            value: role,
          }))}
          size="small"
          style={{ width: '100%' }}
          value={value}
        />
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.user.tokenQuota' }),
      width: 210,
      render: (_, record) => (
        <span>
          {record.usedTokens || 0} /{' '}
          {record.tokenLimit || intl.formatMessage({ id: 'common.unlimited' })}
          <Progress
            percent={percent(record.usedTokens, record.tokenLimit)}
            showInfo={false}
            size="small"
          />
        </span>
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.user.requestQuota' }),
      width: 190,
      render: (_, record) => (
        <span>
          {record.usedRequests || 0} /{' '}
          {record.requestLimit ||
            intl.formatMessage({ id: 'common.unlimited' })}
          <Progress
            percent={percent(record.usedRequests, record.requestLimit)}
            showInfo={false}
            size="small"
            strokeColor="#52c41a"
          />
        </span>
      ),
    },
    {
      title: intl.formatMessage({ id: 'common.status' }),
      width: 90,
      render: (_, record) => (
        <Tag color={record.enable ? 'success' : 'default'}>
          {intl.formatMessage({
            id: record.enable ? 'common.enabled' : 'common.disabled',
          })}
        </Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'common.actions' }),
      width: 270,
      render: (_, record) => (
        <Space>
          <Button
            icon={<DashboardOutlined />}
            onClick={() => history.push(`/system/users/${record.id}/dashboard`)}
            size="small"
            type="link"
          >
            {intl.formatMessage({ id: 'system.user.dashboard' })}
          </Button>
          <Button
            icon={<EditOutlined />}
            onClick={() => showQuota(record)}
            size="small"
            type="link"
          >
            {intl.formatMessage({ id: 'system.user.quota' })}
          </Button>
          <Button
            danger={record.enable}
            onClick={() => void toggle(record)}
            size="small"
            type="link"
          >
            {intl.formatMessage({
              id: record.enable ? 'common.disable' : 'common.enable',
            })}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <SystemPage
      extra={
        <Button icon={<ReloadOutlined />} onClick={() => void load()}>
          {intl.formatMessage({ id: 'common.refresh' })}
        </Button>
      }
      pageKey="users"
    >
      <Table
        columns={columns}
        dataSource={data}
        loading={loading}
        onChange={(page) => void load(page.current, page.pageSize)}
        pagination={pagination}
        rowKey="id"
        scroll={{ x: 1080 }}
      />
      <Modal
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        confirmLoading={saving}
        okText={intl.formatMessage({ id: 'common.save' })}
        onCancel={() => setEditing(undefined)}
        onOk={() => void saveQuota()}
        open={Boolean(editing)}
        title={intl.formatMessage(
          { id: 'system.user.quotaTitle' },
          { name: editing?.nickname || editing?.username },
        )}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            extra={intl.formatMessage({ id: 'tokens.zeroUnlimited' })}
            label={intl.formatMessage({ id: 'system.user.tokenQuota' })}
            name="tokenLimit"
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            extra={intl.formatMessage({ id: 'tokens.zeroUnlimited' })}
            label={intl.formatMessage({ id: 'system.user.requestQuota' })}
            name="requestLimit"
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </SystemPage>
  );
};

export default UsersPage;
