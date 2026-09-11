import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  Button,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { listChannels } from '@/services/channel_management.api';
import type {
  AIModelDetail,
  ChannelDetail,
  ModelRouteDetail,
  ModelRouteInput,
} from '@/services/control_plane.d';
import { listModels } from '@/services/model_management.api';
import {
  createRoute,
  deleteRoute,
  listRoutes,
  updateRoute,
} from '@/services/route_management.api';
import { EnabledTag, HealthTag, isConflictError, SystemPage } from '../shared';
import styles from './index.module.less';

const allChannels = 'all';

const RoutesPage = () => {
  const intl = useIntl();
  const [data, setData] = useState<ModelRouteDetail[]>([]);
  const [models, setModels] = useState<AIModelDetail[]>([]);
  const [channels, setChannels] = useState<ChannelDetail[]>([]);
  const [activeChannel, setActiveChannel] = useState(allChannels);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState<ModelRouteDetail>();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<ModelRouteInput>();

  const selectedChannel = channels.find(
    (channel) => channel.id === activeChannel,
  );

  const loadLookups = useCallback(async () => {
    try {
      const [modelResult, channelResult] = await Promise.all([
        listModels({}),
        listChannels({}),
      ]);
      setModels(modelResult.data || []);
      setChannels(channelResult.data || []);
    } catch {
      message.error(intl.formatMessage({ id: 'system.loadFailed' }));
    }
  }, [intl]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listRoutes({
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
    void loadLookups();
  }, [loadLookups]);

  useEffect(() => {
    void load();
  }, [load]);

  const showForm = (record?: ModelRouteDetail) => {
    setEditing(record);
    form.setFieldsValue({
      modelId: record?.modelId || models[0]?.id || '',
      channelId:
        record?.channelId ||
        (activeChannel === allChannels ? channels[0]?.id : activeChannel) ||
        '',
      upstreamModel: record?.upstreamModel || '',
      priority: record?.priority ?? 0,
      weight: record?.weight || 1,
      status: record?.status || 'enabled',
    });
    setOpen(true);
  };

  const save = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      if (editing) {
        await updateRoute(
          { id: editing.id },
          { ...values, version: editing.version || 0 },
        );
      } else {
        await createRoute({ ...values, version: 0 });
      }
      setOpen(false);
      await Promise.all([load(), loadLookups()]);
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
      await deleteRoute({ id });
      await Promise.all([load(), loadLookups()]);
    } catch {
      message.error(intl.formatMessage({ id: 'system.deleteFailed' }));
    }
  };

  const columns = (() => {
    const result: ColumnsType<ModelRouteDetail> = [
      {
        title: intl.formatMessage({ id: 'system.route.model' }),
        dataIndex: 'modelName',
        render: (value: string) => <strong>{value}</strong>,
      },
      {
        title: intl.formatMessage({ id: 'system.route.upstream' }),
        dataIndex: 'upstreamModel',
        render: (value: string) => (
          <code className={styles.upstream}>{value}</code>
        ),
      },
    ];
    if (activeChannel === allChannels) {
      result.push({
        title: intl.formatMessage({ id: 'system.route.channel' }),
        render: (_, record) => (
          <span className={styles.channelCell}>
            <strong>{record.channelName}</strong>
            <Tag>{record.providerName}</Tag>
          </span>
        ),
      });
    }
    result.push(
      {
        title: intl.formatMessage({ id: 'system.route.priorityWeight' }),
        width: 130,
        render: (_, record) =>
          `${record.priority || 0} / ${record.weight || 1}`,
      },
      {
        title: intl.formatMessage({ id: 'common.status' }),
        dataIndex: 'status',
        width: 90,
        render: (value: string) => <EnabledTag status={value} />,
      },
      {
        title: intl.formatMessage({ id: 'common.actions' }),
        width: 120,
        render: (_, record) => (
          <Space>
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
              <Button
                danger
                icon={<DeleteOutlined />}
                size="small"
                type="text"
              />
            </Popconfirm>
          </Space>
        ),
      },
    );
    return result;
  })() satisfies ColumnsType<ModelRouteDetail>;

  const tabItems = useMemo(
    () => [
      {
        key: allChannels,
        label: (
          <span className={styles.tabLabel}>
            <span className={styles.allDot} />
            {intl.formatMessage({ id: 'system.routes.all' })}
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
            <span className={styles.routeCount}>{channel.routeCount || 0}</span>
          </span>
        ),
      })),
    ],
    [channels, intl],
  );

  return (
    <SystemPage
      extra={
        <Button
          disabled={!models.length || !channels.length}
          icon={<PlusOutlined />}
          onClick={() => showForm()}
          type="primary"
        >
          {intl.formatMessage({ id: 'system.routes.create' })}
        </Button>
      }
      pageKey="routes"
    >
      <div className={styles.channelRail}>
        <div className={styles.railHeading}>
          <span>{intl.formatMessage({ id: 'system.routes.catalog' })}</span>
          <small>
            {intl.formatMessage({ id: 'system.routes.catalogHint' })}
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
                ? 'system.routes.empty'
                : 'system.routes.channelEmpty',
          }),
        }}
        pagination={false}
        rowKey="id"
        scroll={{ x: activeChannel === allChannels ? 820 : 680 }}
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
          id: editing ? 'system.routes.edit' : 'system.routes.create',
        })}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label={intl.formatMessage({ id: 'system.route.model' })}
            name="modelId"
            rules={[
              {
                required: true,
                message: intl.formatMessage({ id: 'common.required' }),
              },
            ]}
          >
            <Select
              options={models.map((model) => ({
                label: model.name,
                value: model.id,
              }))}
            />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'system.route.channel' })}
            name="channelId"
            rules={[
              {
                required: true,
                message: intl.formatMessage({ id: 'common.required' }),
              },
            ]}
          >
            <Select
              options={channels.map((channel) => ({
                label: `${channel.providerName} / ${channel.name}`,
                value: channel.id,
              }))}
            />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'system.route.upstream' })}
            name="upstreamModel"
            rules={[
              {
                required: true,
                message: intl.formatMessage({ id: 'common.required' }),
              },
            ]}
          >
            <Input placeholder="qwen3-coder-480b-a35b-instruct" />
          </Form.Item>
          <Space align="start">
            <Form.Item
              extra={intl.formatMessage({ id: 'system.route.priorityHint' })}
              label={intl.formatMessage({ id: 'system.channel.priority' })}
              name="priority"
            >
              <InputNumber min={0} />
            </Form.Item>
            <Form.Item
              extra={intl.formatMessage({ id: 'system.route.weightHint' })}
              label={intl.formatMessage({ id: 'system.channel.weight' })}
              name="weight"
            >
              <InputNumber min={1} />
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
    </SystemPage>
  );
};

export default RoutesPage;
