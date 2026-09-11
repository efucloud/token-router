import { PlayCircleOutlined } from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  Alert,
  Button,
  Descriptions,
  Divider,
  Form,
  Input,
  InputNumber,
  message,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { runModelDiagnostic } from '@/services/diagnostic_management.api';
import { listModels } from '@/services/model_management.api';
import type {
  AIModelDetail,
  ModelDiagnosticAttempt,
  ModelDiagnosticInput,
  ModelDiagnosticResult,
} from '@/services/control_plane.d';
import { SystemPage } from '../shared';

const DiagnosticsPage = () => {
  const intl = useIntl();
  const [form] = Form.useForm<ModelDiagnosticInput>();
  const [models, setModels] = useState<AIModelDetail[]>([]);
  const [loadingModels, setLoadingModels] = useState(true);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<ModelDiagnosticResult>();

  const loadModels = useCallback(async () => {
    setLoadingModels(true);
    try {
      const response = await listModels({});
      setModels(
        (response.data || []).filter(
          (model) => model.modality === 'chat' && (model.routeCount || 0) > 0,
        ),
      );
    } catch {
      message.error(intl.formatMessage({ id: 'system.loadFailed' }));
    } finally {
      setLoadingModels(false);
    }
  }, [intl]);

  useEffect(() => {
    void loadModels();
  }, [loadModels]);

  const run = async () => {
    const values = await form.validateFields();
    setRunning(true);
    setResult(undefined);
    try {
      setResult(await runModelDiagnostic(values));
    } catch {
      message.error(intl.formatMessage({ id: 'system.diagnostics.failed' }));
    } finally {
      setRunning(false);
    }
  };

  const attemptColumns = useMemo<ColumnsType<ModelDiagnosticAttempt>>(
    () => [
      { title: '#', dataIndex: 'sequence', width: 56 },
      {
        title: intl.formatMessage({ id: 'system.route.channel' }),
        render: (_, record) => (
          <span>
            {record.channelName || record.channelId}
            <br />
            <Typography.Text type="secondary">
              {record.providerName}
            </Typography.Text>
          </span>
        ),
      },
      {
        title: intl.formatMessage({ id: 'system.route.upstream' }),
        dataIndex: 'upstreamModel',
      },
      { title: 'HTTP', dataIndex: 'httpStatus', width: 80 },
      {
        title: intl.formatMessage({ id: 'common.duration' }),
        dataIndex: 'durationMs',
        width: 110,
        render: (value: number) => `${value || 0} ms`,
      },
      {
        title: intl.formatMessage({ id: 'admin.error' }),
        render: (_, record) =>
          record.errorSummary
            ? `${record.errorCode || ''} ${record.errorSummary}`.trim()
            : '-',
      },
    ],
    [intl],
  );

  return (
    <SystemPage pageKey="diagnostics">
      <div style={{ padding: 16 }}>
        <Form
          form={form}
          initialValues={{ maxOutputTokens: 512 }}
          layout="vertical"
        >
          <Space align="start" size="middle" style={{ display: 'flex' }} wrap>
            <Form.Item
              label={intl.formatMessage({ id: 'system.diagnostics.model' })}
              name="modelId"
              rules={[{ required: true }]}
              style={{ minWidth: 280 }}
            >
              <Select
                loading={loadingModels}
                options={models.map((model) => ({
                  label: `${model.displayName || model.name} (${model.name})`,
                  value: model.id,
                }))}
                showSearch
                optionFilterProp="label"
              />
            </Form.Item>
            <Form.Item
              label={intl.formatMessage({
                id: 'system.diagnostics.maxOutput',
              })}
              name="maxOutputTokens"
              rules={[{ required: true }]}
            >
              <InputNumber max={32768} min={1} style={{ width: 160 }} />
            </Form.Item>
          </Space>
          <Form.Item
            label={intl.formatMessage({ id: 'system.diagnostics.prompt' })}
            name="prompt"
            rules={[{ required: true, max: 16000 }]}
          >
            <Input.TextArea
              autoSize={{ minRows: 4, maxRows: 10 }}
              placeholder={intl.formatMessage({
                id: 'system.diagnostics.promptPlaceholder',
              })}
            />
          </Form.Item>
          <Button
            icon={<PlayCircleOutlined />}
            loading={running}
            onClick={() => void run()}
            type="primary"
          >
            {intl.formatMessage({ id: 'system.diagnostics.run' })}
          </Button>
        </Form>

        {result ? (
          <>
            <Divider />
            <Alert
              message={result.message}
              showIcon
              type={result.success ? 'success' : 'error'}
            />
            <Typography.Title level={5} style={{ marginTop: 20 }}>
              {intl.formatMessage({ id: 'system.diagnostics.telemetry' })}
            </Typography.Title>
            <Descriptions
              bordered
              column={{ xs: 1, sm: 2, lg: 4 }}
              size="small"
            >
              <Descriptions.Item label="Request ID" span={2}>
                <Typography.Text copyable>{result.requestId}</Typography.Text>
              </Descriptions.Item>
              <Descriptions.Item
                label={intl.formatMessage({ id: 'system.diagnostics.ttft' })}
              >
                {result.ttftMs || 0} ms
              </Descriptions.Item>
              <Descriptions.Item
                label={intl.formatMessage({ id: 'common.duration' })}
              >
                {result.durationMs || 0} ms
              </Descriptions.Item>
              <Descriptions.Item
                label={intl.formatMessage({
                  id: 'system.diagnostics.finalChannel',
                })}
                span={2}
              >
                {result.finalChannelName || result.finalChannelId || '-'}
              </Descriptions.Item>
              <Descriptions.Item
                label={intl.formatMessage({ id: 'common.tokenUsage' })}
                span={2}
              >
                {result.promptTokens || 0} + {result.completionTokens || 0} ={' '}
                {result.totalTokens || 0}
                {result.estimated ? (
                  <Tag style={{ marginInlineStart: 8 }}>
                    {intl.formatMessage({
                      id: 'system.diagnostics.estimated',
                    })}
                  </Tag>
                ) : null}
              </Descriptions.Item>
            </Descriptions>

            <Typography.Title level={5} style={{ marginTop: 20 }}>
              {intl.formatMessage({ id: 'system.diagnostics.output' })}
            </Typography.Title>
            <Typography.Paragraph
              style={{ marginBottom: 0, whiteSpace: 'pre-wrap' }}
            >
              {result.outputText ||
                intl.formatMessage({ id: 'system.diagnostics.noOutput' })}
            </Typography.Paragraph>

            <Typography.Title level={5} style={{ marginTop: 20 }}>
              {intl.formatMessage({ id: 'system.diagnostics.attempts' })}
            </Typography.Title>
            <Table
              columns={attemptColumns}
              dataSource={result.attempts || []}
              pagination={false}
              rowKey={(record) => `${record.sequence}-${record.routeId}`}
              size="small"
              scroll={{ x: 760 }}
            />
          </>
        ) : null}
      </div>
    </SystemPage>
  );
};

export default DiagnosticsPage;
