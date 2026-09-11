import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  Button,
  Form,
  Input,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useState } from 'react';
import type { ProviderDetail, ProviderInput } from '@/services/control_plane.d';
import {
  createProvider,
  deleteProvider,
  listProviders,
  updateProvider,
} from '@/services/provider_management.api';
import { EnabledTag, isConflictError, SystemPage } from '../shared';

const ProvidersPage = () => {
  const intl = useIntl();
  const [data, setData] = useState<ProviderDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState<ProviderDetail>();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<ProviderInput>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listProviders({});
      setData(result.data || []);
    } catch {
      message.error(intl.formatMessage({ id: 'system.loadFailed' }));
    } finally {
      setLoading(false);
    }
  }, [intl]);

  useEffect(() => {
    void load();
  }, [load]);

  const showForm = (record?: ProviderDetail) => {
    setEditing(record);
    form.setFieldsValue({
      name: record?.name || '',
      type: record?.type || 'openai-compatible',
      status: record?.status || 'enabled',
      config: record?.config || {},
    });
    setOpen(true);
  };

  const save = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      if (editing) {
        await updateProvider(
          { id: editing.id },
          { ...values, version: editing.version || 0 },
        );
      } else {
        await createProvider({ ...values, version: 0 });
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
      await deleteProvider({ id });
      await load();
    } catch {
      message.error(intl.formatMessage({ id: 'system.deleteReferenced' }));
    }
  };

  const columns: ColumnsType<ProviderDetail> = [
    {
      title: intl.formatMessage({ id: 'system.provider.name' }),
      dataIndex: 'name',
      render: (value: string) => <strong>{value}</strong>,
    },
    {
      title: intl.formatMessage({ id: 'system.provider.adapter' }),
      dataIndex: 'type',
      render: (value: string) => <Tag>{value}</Tag>,
    },
    {
      title: intl.formatMessage({ id: 'system.provider.channels' }),
      dataIndex: 'channelCount',
      width: 100,
    },
    {
      title: intl.formatMessage({ id: 'common.status' }),
      dataIndex: 'status',
      width: 100,
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
          icon={<PlusOutlined />}
          onClick={() => showForm()}
          type="primary"
        >
          {intl.formatMessage({ id: 'system.providers.create' })}
        </Button>
      }
      pageKey="providers"
    >
      <Table
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={false}
        rowKey="id"
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
          id: editing ? 'system.providers.edit' : 'system.providers.create',
        })}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label={intl.formatMessage({ id: 'system.provider.name' })}
            name="name"
            rules={[
              {
                required: true,
                message: intl.formatMessage({ id: 'common.required' }),
              },
            ]}
          >
            <Input placeholder="Bailian Production" />
          </Form.Item>
          <Form.Item
            extra={intl.formatMessage({ id: 'system.provider.adapterHint' })}
            label={intl.formatMessage({ id: 'system.provider.adapter' })}
            name="type"
          >
            <Select
              options={[
                { label: 'OpenAI Compatible', value: 'openai-compatible' },
                {
                  label: 'Azure OpenAI',
                  value: 'azure-openai',
                  disabled: true,
                },
                { label: 'Anthropic', value: 'anthropic', disabled: true },
                { label: 'Gemini', value: 'gemini', disabled: true },
              ]}
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

export default ProvidersPage;
