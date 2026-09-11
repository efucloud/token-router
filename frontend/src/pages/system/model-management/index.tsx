import {
  CloudSyncOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  Button,
  Col,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  listChannels,
  syncChannelModels,
} from '@/services/channel_management.api';
import type {
  AIModelDetail,
  AIModelInput,
  ChannelDetail,
} from '@/services/control_plane.d';
import {
  createModel,
  deleteModel,
  listModels,
  updateModel,
} from '@/services/model_management.api';
import { EnabledTag, HealthTag, isConflictError, SystemPage } from '../shared';
import styles from './index.module.less';

const allChannels = 'all';
const capabilities = ['streaming', 'tools', 'reasoning', 'vision', 'json-mode'];

const ModelsPage = () => {
  const intl = useIntl();
  const [data, setData] = useState<AIModelDetail[]>([]);
  const [channels, setChannels] = useState<ChannelDetail[]>([]);
  const [activeChannel, setActiveChannel] = useState(allChannels);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [editing, setEditing] = useState<AIModelDetail>();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<AIModelInput>();

  const selectedChannel = channels.find(
    (channel) => channel.id === activeChannel,
  );

  const loadChannels = useCallback(async () => {
    try {
      const result = await listChannels({});
      setChannels(result.data || []);
    } catch {
      message.error(intl.formatMessage({ id: 'system.loadFailed' }));
    }
  }, [intl]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listModels({
        channelId: activeChannel === allChannels ? undefined : activeChannel,
      });
      setData(result.data || []);
    } catch {
      message.error(intl.formatMessage({ id: 'system.loadFailed' }));
    } finally {
      setLoading(false);
    }
  }, [activeChannel, intl]);

  useEffect(() => {
    void loadChannels();
  }, [loadChannels]);

  useEffect(() => {
    void load();
  }, [load]);

  const showForm = (record?: AIModelDetail) => {
    setEditing(record);
    form.setFieldsValue(
      record
        ? {
            name: record.name || '',
            displayName: record.displayName || '',
            description: record.description || '',
            modality: record.modality || 'chat',
            contextWindow: record.contextWindow || 0,
            maxOutputTokens: record.maxOutputTokens || 0,
            capabilities: record.capabilities || [],
            status: record.status || 'active',
          }
        : {
            name: '',
            displayName: '',
            description: '',
            modality: 'chat',
            contextWindow: 0,
            maxOutputTokens: 0,
            capabilities: ['streaming'],
            status: 'active',
          },
    );
    setOpen(true);
  };

  const save = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      if (editing) {
        await updateModel(
          { id: editing.id },
          { ...values, version: editing.version || 0 },
        );
      } else {
        await createModel({ ...values, version: 0 });
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
      await deleteModel({ id });
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'system.deleteReferenced' }));
    }
  };

  const sync = async () => {
    if (!selectedChannel) return;
    setSyncing(true);
    try {
      const result = await syncChannelModels({ id: selectedChannel.id }, {});
      await Promise.all([load(), loadChannels()]);
      message.success(
        intl.formatMessage(
          { id: 'system.models.syncSuccess' },
          {
            discovered: result.discovered || 0,
            modelsCreated: result.modelsCreated || 0,
            routesCreated: result.routesCreated || 0,
            routesDeleted: result.routesDeleted || 0,
            modelsDeleted: result.modelsDeleted || 0,
          },
        ),
      );
    } catch {
      message.error(intl.formatMessage({ id: 'system.models.syncFailed' }));
    } finally {
      setSyncing(false);
    }
  };

  const columns = (() => {
    const result: ColumnsType<AIModelDetail> = [
      {
        title: intl.formatMessage({ id: 'system.model.name' }),
        dataIndex: 'name',
        render: (value: string, record) => (
          <span className={styles.modelIdentity}>
            <strong>{record.displayName || value}</strong>
            <code>{value}</code>
          </span>
        ),
      },
    ];
    if (activeChannel !== allChannels) {
      result.push({
        title: intl.formatMessage({ id: 'system.model.upstream' }),
        dataIndex: 'upstreamModel',
        render: (value: string) => (
          <code className={styles.upstream}>{value}</code>
        ),
      });
    }
    result.push(
      {
        title: intl.formatMessage({ id: 'system.model.modality' }),
        dataIndex: 'modality',
        width: 110,
        render: (value: string) => <Tag>{value}</Tag>,
      },
      {
        title: intl.formatMessage({ id: 'system.model.context' }),
        width: 150,
        render: (_, record) =>
          `${record.contextWindow || '-'} / ${record.maxOutputTokens || '-'}`,
      },
      {
        title: intl.formatMessage({ id: 'system.model.capabilities' }),
        dataIndex: 'capabilities',
        render: (values: string[]) =>
          values?.length
            ? values.map((value) => <Tag key={value}>{value}</Tag>)
            : '-',
      },
      activeChannel === allChannels
        ? {
            title: intl.formatMessage({ id: 'system.model.routes' }),
            dataIndex: 'routeCount',
            width: 90,
          }
        : {
            title: intl.formatMessage({ id: 'system.model.routeStatus' }),
            dataIndex: 'routeStatus',
            width: 110,
            render: (value: string) => <EnabledTag status={value} />,
          },
      {
        title: intl.formatMessage({ id: 'common.status' }),
        dataIndex: 'status',
        width: 90,
        render: (value: string) => <EnabledTag status={value} />,
      },
      {
        title: intl.formatMessage({ id: 'common.actions' }),
        width: activeChannel === allChannels ? 120 : 72,
        render: (_, record) => (
          <Space>
            <Button
              icon={<EditOutlined />}
              onClick={() => showForm(record)}
              size="small"
              type="text"
            />
            {activeChannel === allChannels ? (
              <Popconfirm
                cancelText={intl.formatMessage({ id: 'common.cancel' })}
                description={intl.formatMessage({ id: 'system.deleteHint' })}
                okButtonProps={{ danger: true }}
                okText={intl.formatMessage({ id: 'common.delete' })}
                onConfirm={() => void remove(record.id)}
                title={intl.formatMessage({ id: 'system.deleteTitle' })}
              >
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  size="small"
                  type="text"
                />
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    );
    return result;
  })() satisfies ColumnsType<AIModelDetail>;

  const tabItems = useMemo(
    () => [
      {
        key: allChannels,
        label: (
          <span className={styles.tabLabel}>
            <span className={styles.allDot} />
            {intl.formatMessage({ id: 'system.models.all' })}
          </span>
        ),
      },
      ...channels.map((channel) => ({
        key: channel.id,
        label: (
          <span className={styles.tabLabel}>
            <span
              className={`${styles.healthDot} ${
                channel.healthStatus === 'healthy' ? styles.healthy : ''
              }`}
            />
            <span>{channel.name}</span>
            <span className={styles.modelCount}>{channel.routeCount || 0}</span>
          </span>
        ),
      })),
    ],
    [channels, intl],
  );

  return (
    <SystemPage
      extra={
        <Space>
          {selectedChannel ? (
            <Button
              icon={<CloudSyncOutlined />}
              loading={syncing}
              onClick={() => void sync()}
              type="primary"
            >
              {intl.formatMessage({ id: 'system.models.sync' })}
            </Button>
          ) : (
            <Button
              icon={<PlusOutlined />}
              onClick={() => showForm()}
              type="primary"
            >
              {intl.formatMessage({ id: 'system.models.create' })}
            </Button>
          )}
        </Space>
      }
      pageKey="models"
    >
      <div className={styles.channelRail}>
        <div className={styles.railHeading}>
          <span>{intl.formatMessage({ id: 'system.models.catalog' })}</span>
          <small>
            {intl.formatMessage({ id: 'system.models.catalogHint' })}
          </small>
        </div>
        <Tabs
          activeKey={activeChannel}
          items={tabItems}
          onChange={setActiveChannel}
          tabBarGutter={8}
        />
      </div>
      {selectedChannel ? (
        <div className={styles.channelContext}>
          <div>
            <span className={styles.provider}>
              {selectedChannel.providerName}
            </span>
            <strong>{selectedChannel.name}</strong>
          </div>
          <div className={styles.channelMeta}>
            <HealthTag status={selectedChannel.healthStatus} />
            <span>{selectedChannel.baseUrl}</span>
          </div>
          <p>{intl.formatMessage({ id: 'system.models.syncHint' })}</p>
        </div>
      ) : null}
      <Table
        columns={columns}
        dataSource={data}
        loading={loading}
        locale={{
          emptyText: intl.formatMessage({
            id:
              activeChannel === allChannels
                ? 'system.models.empty'
                : 'system.models.channelEmpty',
          }),
        }}
        pagination={false}
        rowKey={(record) => record.routeId || record.id}
        scroll={{ x: activeChannel === allChannels ? 960 : 1120 }}
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
          id: editing ? 'system.models.edit' : 'system.models.create',
        })}
        width={640}
      >
        <Form form={form} layout="vertical">
          <Space align="start" style={{ display: 'flex' }}>
            <Form.Item
              label={intl.formatMessage({ id: 'system.model.name' })}
              name="name"
              rules={[
                {
                  required: true,
                  message: intl.formatMessage({ id: 'common.required' }),
                },
              ]}
              style={{ flex: 1 }}
            >
              <Input placeholder="qwen-coder" />
            </Form.Item>
            <Form.Item
              label={intl.formatMessage({ id: 'system.model.displayName' })}
              name="displayName"
              rules={[
                {
                  required: true,
                  message: intl.formatMessage({ id: 'common.required' }),
                },
              ]}
              style={{ flex: 1 }}
            >
              <Input />
            </Form.Item>
          </Space>
          <Form.Item
            label={intl.formatMessage({ id: 'system.model.description' })}
            name="description"
          >
            <Input.TextArea maxLength={1000} rows={2} />
          </Form.Item>
          <Row gutter={16}>
            <Col sm={8} xs={24}>
              <Form.Item
                label={intl.formatMessage({ id: 'system.model.modality' })}
                name="modality"
              >
                <Select
                  options={[
                    { label: 'Chat', value: 'chat' },
                    { label: 'Embedding', value: 'embedding' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col sm={8} xs={24}>
              <Form.Item
                label={intl.formatMessage({ id: 'system.model.contextWindow' })}
                name="contextWindow"
              >
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col sm={8} xs={24}>
              <Form.Item
                label={intl.formatMessage({
                  id: 'system.model.maxOutputTokens',
                })}
                name="maxOutputTokens"
              >
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            label={intl.formatMessage({ id: 'system.model.capabilities' })}
            name="capabilities"
          >
            <Select
              mode="tags"
              options={capabilities.map((value) => ({ label: value, value }))}
            />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'common.status' })}
            name="status"
          >
            <Select
              options={[
                {
                  label: intl.formatMessage({ id: 'common.enabled' }),
                  value: 'active',
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
    </SystemPage>
  );
};

export default ModelsPage;
