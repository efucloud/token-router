import {
  ApiOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  Button,
  Descriptions,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import {
  createChannel,
  deleteChannel,
  importChannelModels,
  listChannels,
  testChannel,
  updateChannel,
} from '@/services/channel_management.api';
import type {
  ChannelDetail,
  ChannelInput,
  ChannelTestResult,
  ProviderDetail,
} from '@/services/control_plane.d';
import { listProviders } from '@/services/provider_management.api';
import {
  EnabledTag,
  formatTime,
  HealthTag,
  isConflictError,
  SystemPage,
} from '../shared';

const ChannelsPage = () => {
  const intl = useIntl();
  const [data, setData] = useState<ChannelDetail[]>([]);
  const [providers, setProviders] = useState<ProviderDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testingID, setTestingID] = useState('');
  const [testedChannel, setTestedChannel] = useState<ChannelDetail>();
  const [testResult, setTestResult] = useState<ChannelTestResult>();
  const [selectedModels, setSelectedModels] = useState<string[]>([]);
  const [importing, setImporting] = useState(false);
  const [editing, setEditing] = useState<ChannelDetail>();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<ChannelInput>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [channelsResult, providersResult] = await Promise.all([
        listChannels({}),
        listProviders({}),
      ]);
      setData(channelsResult.data || []);
      setProviders(providersResult.data || []);
    } catch {
      message.error(intl.formatMessage({ id: 'system.loadFailed' }));
    } finally {
      setLoading(false);
    }
  }, [intl]);

  useEffect(() => {
    void load();
  }, [load]);

  const showForm = (record?: ChannelDetail) => {
    setEditing(record);
    form.setFieldsValue({
      providerId: record?.providerId || providers[0]?.id || '',
      name: record?.name || '',
      baseUrl: record?.baseUrl || '',
      apiKey: '',
      priority: record?.priority ?? 0,
      weight: record?.weight || 1,
      status: record?.status || 'enabled',
      timeoutSeconds: record?.timeoutSeconds || 0,
      maxConcurrency: record?.maxConcurrency || 0,
    });
    setOpen(true);
  };

  const save = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      if (editing) {
        await updateChannel(
          { id: editing.id },
          { ...values, version: editing.version || 0 },
        );
      } else {
        await createChannel({ ...values, version: 0 });
      }
      setOpen(false);
      await load();
      message.success(intl.formatMessage({ id: 'system.saved' }));
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: isConflictError(error)
            ? 'system.saveConflict'
            : 'system.saveFailed',
        }),
      );
    } finally {
      setSaving(false);
    }
  };

  const remove = async (id: string) => {
    try {
      await deleteChannel({ id });
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'system.deleteReferenced' }));
    }
  };

  const test = async (record: ChannelDetail) => {
    setTestingID(record.id);
    try {
      const result = await testChannel({ id: record.id });
      setTestedChannel(record);
      setTestResult(result);
      setSelectedModels(result.success ? result.models || [] : []);
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'system.channel.testFailed' }));
    } finally {
      setTestingID('');
    }
  };

  const importModels = async () => {
    if (!testedChannel || !selectedModels.length) return;
    setImporting(true);
    try {
      const result = await importChannelModels(
        { id: testedChannel.id },
        { models: selectedModels },
      );
      message.success(
        intl.formatMessage(
          { id: 'system.channel.imported' },
          {
            models: result.modelsCreated || 0,
            routes: result.routesCreated || 0,
            skipped: result.skipped || 0,
          },
        ),
      );
      setTestResult(undefined);
      setTestedChannel(undefined);
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'system.channel.importFailed' }));
    } finally {
      setImporting(false);
    }
  };

  const columns: ColumnsType<ChannelDetail> = [
    {
      title: intl.formatMessage({ id: 'system.channel.name' }),
      render: (_, record) => (
        <span>
          <strong>{record.name}</strong>
          <br />
          <Typography.Text copyable={{ text: record.baseUrl }} type="secondary">
            {record.baseUrl}
          </Typography.Text>
        </span>
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.channel.provider' }),
      dataIndex: 'providerName',
      width: 150,
    },
    {
      title: intl.formatMessage({ id: 'system.channel.health' }),
      width: 150,
      render: (_, record) => (
        <span>
          <HealthTag status={record.healthStatus} />
          <br />
          <Typography.Text type="secondary">
            {record.lastCheckedAt
              ? `${record.lastLatencyMs || 0} ms · ${formatTime(record.lastCheckedAt)}`
              : intl.formatMessage({ id: 'system.channel.notChecked' })}
          </Typography.Text>
        </span>
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.channel.routing' }),
      width: 140,
      render: (_, record) => `${record.priority || 0} / ${record.weight || 1}`,
    },
    {
      title: intl.formatMessage({ id: 'system.channel.credential' }),
      dataIndex: 'credentialConfigured',
      width: 100,
      render: (value: boolean) => (
        <Tag color={value ? 'success' : 'default'}>
          {intl.formatMessage({
            id: value
              ? 'system.channel.configured'
              : 'system.channel.noCredential',
          })}
        </Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'common.status' }),
      dataIndex: 'status',
      width: 90,
      render: (value: string) => <EnabledTag status={value} />,
    },
    {
      title: intl.formatMessage({ id: 'common.actions' }),
      fixed: 'right',
      width: 170,
      render: (_, record) => (
        <Space>
          <Button
            icon={<ApiOutlined />}
            loading={testingID === record.id}
            onClick={() => void test(record)}
            size="small"
            type="link"
          >
            {intl.formatMessage({ id: 'system.channel.test' })}
          </Button>
          <Button
            icon={<EditOutlined />}
            onClick={() => showForm(record)}
            size="small"
            type="text"
          />
          <Popconfirm
            cancelText={intl.formatMessage({ id: 'common.cancel' })}
            description={intl.formatMessage({ id: 'system.deleteHint' })}
            okButtonProps={{ danger: true }}
            okText={intl.formatMessage({ id: 'common.delete' })}
            onConfirm={() => void remove(record.id)}
            title={intl.formatMessage({ id: 'system.deleteTitle' })}
          >
            <Button danger icon={<DeleteOutlined />} size="small" type="text" />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <SystemPage
      extra={
        <Button
          disabled={!providers.length}
          icon={<PlusOutlined />}
          onClick={() => showForm()}
          type="primary"
        >
          {intl.formatMessage({ id: 'system.channels.create' })}
        </Button>
      }
      pageKey="channels"
    >
      <Table
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={false}
        rowKey="id"
        scroll={{ x: 1180 }}
      />
      <Modal
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        confirmLoading={saving}
        destroyOnHidden
        okText={intl.formatMessage({
          id: editing ? 'common.save' : 'common.create',
        })}
        onCancel={() => setOpen(false)}
        onOk={() => void save()}
        open={open}
        title={intl.formatMessage({
          id: editing ? 'system.channels.edit' : 'system.channels.create',
        })}
        width={680}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label={intl.formatMessage({ id: 'system.channel.provider' })}
            name="providerId"
            rules={[
              {
                required: true,
                message: intl.formatMessage({ id: 'common.required' }),
              },
            ]}
          >
            <Select
              options={providers.map((provider) => ({
                label: provider.name,
                value: provider.id,
              }))}
            />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'system.channel.name' })}
            name="name"
            rules={[
              {
                required: true,
                message: intl.formatMessage({ id: 'common.required' }),
              },
            ]}
          >
            <Input placeholder="bailian-default" />
          </Form.Item>
          <Form.Item
            label="Base URL"
            name="baseUrl"
            rules={[
              {
                required: true,
                message: intl.formatMessage({ id: 'common.required' }),
              },
              {
                type: 'url',
                message: intl.formatMessage({ id: 'common.invalidUrl' }),
              },
            ]}
          >
            <Input placeholder="https://dashscope.aliyuncs.com/compatible-mode/v1" />
          </Form.Item>
          <Form.Item
            extra={intl.formatMessage({
              id: editing
                ? 'system.channel.keepCredential'
                : 'system.channel.optionalCredential',
            })}
            label="API Key"
            name="apiKey"
          >
            <Input.Password autoComplete="new-password" />
          </Form.Item>
          <Space align="start" style={{ display: 'flex' }}>
            <Form.Item
              label={intl.formatMessage({ id: 'system.channel.priority' })}
              name="priority"
            >
              <InputNumber min={0} />
            </Form.Item>
            <Form.Item
              label={intl.formatMessage({ id: 'system.channel.weight' })}
              name="weight"
            >
              <InputNumber min={1} />
            </Form.Item>
            <Form.Item
              label={intl.formatMessage({ id: 'system.channel.timeout' })}
              name="timeoutSeconds"
            >
              <InputNumber max={600} min={0} />
            </Form.Item>
            <Form.Item
              label={intl.formatMessage({ id: 'system.channel.concurrency' })}
              name="maxConcurrency"
            >
              <InputNumber min={0} />
            </Form.Item>
          </Space>
          <Form.Item
            label={intl.formatMessage({ id: 'common.status' })}
            name="status"
          >
            <Select
              options={[
                {
                  label: intl.formatMessage({ id: 'common.enabled' }),
                  value: 'enabled',
                },
                {
                  label: intl.formatMessage({ id: 'common.disabled' }),
                  value: 'disabled',
                },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        cancelText={intl.formatMessage({ id: 'common.close' })}
        confirmLoading={importing}
        destroyOnHidden
        okButtonProps={{
          disabled: !testResult?.success || selectedModels.length === 0,
        }}
        okText={intl.formatMessage({ id: 'system.channel.importSelected' })}
        onCancel={() => {
          setTestResult(undefined);
          setTestedChannel(undefined);
        }}
        onOk={() => void importModels()}
        open={Boolean(testResult)}
        title={intl.formatMessage({ id: 'system.channel.testResult' })}
        width={680}
      >
        <Descriptions column={2} size="small" style={{ marginTop: 20 }}>
          <Descriptions.Item
            label={intl.formatMessage({ id: 'common.status' })}
          >
            <HealthTag status={testResult?.success ? 'healthy' : 'unhealthy'} />
          </Descriptions.Item>
          <Descriptions.Item label="HTTP">
            {testResult?.httpStatus || '-'}
          </Descriptions.Item>
          <Descriptions.Item
            label={intl.formatMessage({ id: 'common.duration' })}
          >
            {testResult?.latencyMs || 0} ms
          </Descriptions.Item>
          <Descriptions.Item
            label={intl.formatMessage({ id: 'system.channel.models' })}
          >
            {testResult?.modelCount || 0}
          </Descriptions.Item>
          <Descriptions.Item
            label={intl.formatMessage({ id: 'system.channel.message' })}
            span={2}
          >
            {testResult?.message}
          </Descriptions.Item>
        </Descriptions>
        {testResult?.models?.length ? (
          <Form.Item
            extra={intl.formatMessage({ id: 'system.channel.importHint' })}
            label={intl.formatMessage({ id: 'system.channel.discovered' })}
            style={{ marginTop: 20 }}
          >
            <Select
              mode="multiple"
              onChange={setSelectedModels}
              options={testResult.models.map((model) => ({
                label: model,
                value: model,
              }))}
              value={selectedModels}
            />
          </Form.Item>
        ) : null}
      </Modal>
    </SystemPage>
  );
};

export default ChannelsPage;
