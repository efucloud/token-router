import {
  ApiOutlined,
  CopyOutlined,
  DeleteOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import {
  Button,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Progress,
  Select,
  Space,
  Table,
  Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import {
  createMyAPIToken,
  deleteMyAPIToken,
  listMyAPITokens,
  updateMyAPIToken,
} from '@/services/api_token.api';
import type {
  APITokenCreate,
  APITokenDetail,
  APITokenList,
} from '@/services/api_token.d';
import styles from './index.less';

const usagePercent = (used?: number, limit?: number) =>
  limit ? Math.min(100, Math.round(((used || 0) / limit) * 100)) : 0;

const TokenPage = () => {
  const intl = useIntl();
  const apiBaseUrl =
    typeof window === 'undefined' ? '/v1' : `${window.location.origin}/v1`;
  const [items, setItems] = useState<APITokenDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [open, setOpen] = useState(false);
  const [createdToken, setCreatedToken] = useState('');
  const [form] = Form.useForm<APITokenCreate>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = (await listMyAPITokens()) as APITokenList;
      setItems(result.data || []);
    } catch {
      message.error(intl.formatMessage({ id: 'tokens.loadFailed' }));
    } finally {
      setLoading(false);
    }
  }, [intl.formatMessage]);

  useEffect(() => {
    void load();
  }, [load]);

  const create = async () => {
    const values = await form.validateFields();
    setCreating(true);
    try {
      const result = await createMyAPIToken(values);
      setCreatedToken(result.token || '');
      setOpen(false);
      form.resetFields();
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'tokens.createFailed' }));
    } finally {
      setCreating(false);
    }
  };

  const toggle = async (record: APITokenDetail) => {
    try {
      await updateMyAPIToken(
        { id: record.id },
        {
          name: record.name || '',
          allowedModels: record.allowedModels || [],
          tokenLimit: record.tokenLimit || 0,
          requestLimit: record.requestLimit || 0,
          ipAllowlist: record.ipAllowlist || [],
          rpm: record.rpm || 0,
          tpm: record.tpm || 0,
          maxConcurrency: record.maxConcurrency || 0,
          expiresAt: record.expiresAt,
          status: record.status === 'active' ? 'disabled' : 'active',
        },
      );
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'tokens.updateFailed' }));
    }
  };

  const remove = async (id: string) => {
    try {
      await deleteMyAPIToken({ id });
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'tokens.deleteFailed' }));
    }
  };

  const columns: ColumnsType<APITokenDetail> = [
    {
      title: intl.formatMessage({ id: 'tokens.name' }),
      dataIndex: 'name',
      render: (value: string, record) => (
        <span>
          <strong>{value}</strong>
          <br />
          <code>{record.keyPrefix}••••</code>
        </span>
      ),
    },
    {
      title: intl.formatMessage({ id: 'tokens.modelScope' }),
      dataIndex: 'allowedModels',
      render: (models: string[]) =>
        models?.length ? (
          models.map((model) => <Tag key={model}>{model}</Tag>)
        ) : (
          <Tag>{intl.formatMessage({ id: 'tokens.allModels' })}</Tag>
        ),
    },
    {
      title: intl.formatMessage({ id: 'tokens.accessPolicy' }),
      width: 230,
      render: (_, record) => {
        const policies = [
          record.rpm ? `RPM ${record.rpm}` : '',
          record.tpm ? `TPM ${record.tpm}` : '',
          record.maxConcurrency
            ? `${intl.formatMessage({ id: 'tokens.concurrencyShort' })} ${record.maxConcurrency}`
            : '',
          record.ipAllowlist?.length
            ? intl.formatMessage(
                { id: 'tokens.ipRules' },
                { count: record.ipAllowlist.length },
              )
            : '',
        ].filter(Boolean);
        return policies.length ? (
          <Space size={[0, 4]} wrap>
            {policies.map((policy) => (
              <Tag key={policy}>{policy}</Tag>
            ))}
          </Space>
        ) : (
          <Tag>{intl.formatMessage({ id: 'tokens.unrestricted' })}</Tag>
        );
      },
    },
    {
      title: intl.formatMessage({ id: 'common.tokenUsage' }),
      width: 165,
      render: (_, record) => (
        <div className={styles.usageCell}>
          <span>
            {record.usedTokens || 0} /{' '}
            {record.tokenLimit ||
              intl.formatMessage({ id: 'common.unlimited' })}
          </span>
          <Progress
            percent={usagePercent(record.usedTokens, record.tokenLimit)}
            showInfo={false}
            size="small"
            strokeColor="#1677ff"
          />
        </div>
      ),
    },
    {
      title: intl.formatMessage({ id: 'tokens.requestUsage' }),
      width: 165,
      render: (_, record) => (
        <div className={styles.usageCell}>
          <span>
            {record.usedRequests || 0} /{' '}
            {record.requestLimit ||
              intl.formatMessage({ id: 'common.unlimited' })}
          </span>
          <Progress
            percent={usagePercent(record.usedRequests, record.requestLimit)}
            showInfo={false}
            size="small"
            strokeColor="#52c41a"
          />
        </div>
      ),
    },
    {
      title: intl.formatMessage({ id: 'common.status' }),
      dataIndex: 'status',
      width: 90,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'default'}>
          {intl.formatMessage({
            id: status === 'active' ? 'common.active' : 'common.disabled',
          })}
        </Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'tokens.operation' }),
      width: 170,
      render: (_, record) => (
        <Space>
          <Button onClick={() => void toggle(record)} size="small" type="link">
            {intl.formatMessage({
              id:
                record.status === 'active' ? 'tokens.disable' : 'tokens.enable',
            })}
          </Button>
          <Popconfirm
            cancelText={intl.formatMessage({ id: 'common.cancel' })}
            description={intl.formatMessage({ id: 'tokens.deleteHint' })}
            okButtonProps={{ danger: true }}
            okText={intl.formatMessage({ id: 'common.delete' })}
            onConfirm={() => void remove(record.id)}
            title={intl.formatMessage({ id: 'tokens.deleteTitle' })}
          >
            <Button danger icon={<DeleteOutlined />} size="small" type="text" />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className={styles.eyebrow}>USER CREDENTIALS</span>
          <h1 className={styles.title}>API Key</h1>
          <p className={styles.subtitle}>
            {intl.formatMessage({ id: 'tokens.subtitle' })}
          </p>
        </div>
        <Button
          icon={<PlusOutlined />}
          onClick={() => setOpen(true)}
          type="primary"
        >
          {intl.formatMessage({ id: 'tokens.create' })}
        </Button>
      </header>
      <ProCard className={styles.endpointPanel}>
        <div className={styles.endpointContent}>
          <div className={styles.endpointIcon}>
            <ApiOutlined />
          </div>
          <div className={styles.endpointDetails}>
            <span className={styles.endpointLabel}>
              {intl.formatMessage({ id: 'tokens.endpointTitle' })}
            </span>
            <div className={styles.endpointValueRow}>
              <code className={styles.endpointValue}>{apiBaseUrl}</code>
              <Button
                icon={<CopyOutlined />}
                onClick={() => {
                  void navigator.clipboard.writeText(apiBaseUrl);
                  message.success(
                    intl.formatMessage({ id: 'tokens.endpointCopied' }),
                  );
                }}
                size="small"
              >
                {intl.formatMessage({ id: 'tokens.copyEndpoint' })}
              </Button>
            </div>
            <p className={styles.endpointHint}>
              {intl.formatMessage({ id: 'tokens.endpointHint' })}
            </p>
          </div>
        </div>
      </ProCard>
      <ProCard className={styles.panel}>
        <Table
          columns={columns}
          dataSource={items}
          loading={loading}
          locale={{ emptyText: intl.formatMessage({ id: 'tokens.empty' }) }}
          pagination={false}
          rowKey="id"
          scroll={{ x: 980 }}
        />
      </ProCard>

      <Modal
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        confirmLoading={creating}
        okText={intl.formatMessage({ id: 'tokens.confirmCreate' })}
        onCancel={() => setOpen(false)}
        onOk={() => void create()}
        open={open}
        title={intl.formatMessage({ id: 'tokens.createTitle' })}
      >
        <Form
          form={form}
          initialValues={{
            allowedModels: [],
            ipAllowlist: [],
            maxConcurrency: 0,
            requestLimit: 0,
            rpm: 0,
            tokenLimit: 0,
            tpm: 0,
          }}
          layout="vertical"
        >
          <Form.Item
            label={intl.formatMessage({ id: 'tokens.name' })}
            name="name"
            rules={[
              {
                required: true,
                message: intl.formatMessage({ id: 'tokens.nameRequired' }),
              },
            ]}
          >
            <Input
              maxLength={255}
              placeholder={intl.formatMessage({ id: 'tokens.namePlaceholder' })}
            />
          </Form.Item>
          <Form.Item
            extra={intl.formatMessage({ id: 'tokens.allowlistExtra' })}
            label={intl.formatMessage({ id: 'tokens.allowlist' })}
            name="allowedModels"
          >
            <Select
              mode="tags"
              open={false}
              placeholder={intl.formatMessage({
                id: 'tokens.allowlistPlaceholder',
              })}
            />
          </Form.Item>
          <Form.Item
            extra={intl.formatMessage({ id: 'tokens.ipAllowlistExtra' })}
            label={intl.formatMessage({ id: 'tokens.ipAllowlist' })}
            name="ipAllowlist"
          >
            <Select
              mode="tags"
              open={false}
              placeholder={intl.formatMessage({
                id: 'tokens.ipAllowlistPlaceholder',
              })}
            />
          </Form.Item>
          <Space align="start" style={{ width: '100%' }}>
            <Form.Item
              extra={intl.formatMessage({ id: 'tokens.zeroUnlimited' })}
              label={intl.formatMessage({ id: 'tokens.tokenLimit' })}
              name="tokenLimit"
            >
              <InputNumber min={0} style={{ width: 210 }} />
            </Form.Item>
            <Form.Item
              extra={intl.formatMessage({ id: 'tokens.zeroUnlimited' })}
              label={intl.formatMessage({ id: 'tokens.requestLimit' })}
              name="requestLimit"
            >
              <InputNumber min={0} style={{ width: 210 }} />
            </Form.Item>
          </Space>
          <Space align="start" style={{ width: '100%' }} wrap>
            <Form.Item
              extra={intl.formatMessage({ id: 'tokens.zeroUnlimited' })}
              label={intl.formatMessage({ id: 'tokens.rpm' })}
              name="rpm"
            >
              <InputNumber min={0} style={{ width: 136 }} />
            </Form.Item>
            <Form.Item
              extra={intl.formatMessage({ id: 'tokens.zeroUnlimited' })}
              label={intl.formatMessage({ id: 'tokens.tpm' })}
              name="tpm"
            >
              <InputNumber min={0} style={{ width: 136 }} />
            </Form.Item>
            <Form.Item
              extra={intl.formatMessage({ id: 'tokens.zeroUnlimited' })}
              label={intl.formatMessage({ id: 'tokens.maxConcurrency' })}
              name="maxConcurrency"
            >
              <InputNumber min={0} style={{ width: 136 }} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>

      <Modal
        cancelButtonProps={{ style: { display: 'none' } }}
        okText={intl.formatMessage({ id: 'tokens.saved' })}
        onOk={() => setCreatedToken('')}
        open={Boolean(createdToken)}
        title={intl.formatMessage({ id: 'tokens.createdTitle' })}
      >
        <p>{intl.formatMessage({ id: 'tokens.createdHint' })}</p>
        <code className={styles.tokenValue}>{createdToken}</code>
        <Button
          block
          icon={<CopyOutlined />}
          onClick={() => {
            void navigator.clipboard.writeText(createdToken);
            message.success(intl.formatMessage({ id: 'tokens.copied' }));
          }}
          style={{ marginTop: 14 }}
        >
          {intl.formatMessage({ id: 'tokens.copy' })}
        </Button>
      </Modal>
    </main>
  );
};

export default TokenPage;
