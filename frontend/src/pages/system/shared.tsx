import { PageContainer } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Card, Tag } from 'antd';
import type { ReactNode } from 'react';

export const SystemPage = ({
  pageKey,
  extra,
  children,
}: {
  pageKey: string;
  extra?: ReactNode;
  children: ReactNode;
}) => {
  const intl = useIntl();
  return (
    <PageContainer
      content={intl.formatMessage({ id: `system.${pageKey}.description` })}
      extra={extra}
      title={intl.formatMessage({ id: `system.${pageKey}.title` })}
    >
      <Card styles={{ body: { padding: 0 } }}>{children}</Card>
    </PageContainer>
  );
};

export const EnabledTag = ({ status }: { status?: string }) => {
  const intl = useIntl();
  const enabled = status === 'active' || status === 'enabled';
  return (
    <Tag color={enabled ? 'success' : 'default'}>
      {intl.formatMessage({
        id: enabled ? 'common.enabled' : 'common.disabled',
      })}
    </Tag>
  );
};

export const HealthTag = ({ status }: { status?: string }) => {
  const intl = useIntl();
  const color =
    status === 'healthy'
      ? 'success'
      : status === 'unhealthy'
        ? 'error'
        : status === 'cooldown' || status === 'half_open'
          ? 'warning'
          : 'default';
  return (
    <Tag color={color}>
      {intl.formatMessage({
        defaultMessage: status || 'unknown',
        id: `common.health.${status || 'unknown'}`,
      })}
    </Tag>
  );
};

export const formatTime = (value?: string) =>
  value ? new Date(value).toLocaleString() : '-';

export const isConflictError = (error: unknown) =>
  (error as { response?: { status?: number } })?.response?.status === 409;
