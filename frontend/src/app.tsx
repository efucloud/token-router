import {
  ControlOutlined,
  LogoutOutlined,
  MessageOutlined,
  UserOutlined,
} from '@ant-design/icons';
import type { Settings as LayoutSettings } from '@ant-design/pro-components';
import type { RequestConfig, RunTimeLayoutConfig } from '@umijs/max';
import { FormattedMessage, history, Link, useLocation } from '@umijs/max';
import { Button, Dropdown } from 'antd';
import type { ReactNode } from 'react';
import { LangDropdown } from '@/components/LangDropdown';
import type { AuthedUserInfo } from '@/services/common.d';
import { getUserinfo } from '@/services/oauth.api';
import { clearToken, getToken } from '@/utils/auth';
import defaultSettings from '../config/defaultSettings';

const authenticationPaths = new Set(['/oauth/login', '/oauth/callback']);

const HeaderChatEntry = () => {
  const location = useLocation();
  const active = location.pathname === '/personal/chat';

  return (
    <Button
      aria-current={active ? 'page' : undefined}
      icon={<MessageOutlined />}
      onClick={() => history.push('/personal/chat')}
      style={{
        height: 40,
        paddingInline: 14,
        color: active ? '#1677ff' : '#595959',
        background: active ? '#e6f4ff' : 'transparent',
        fontWeight: active ? 600 : 400,
      }}
      type="text"
    >
      <FormattedMessage defaultMessage="模型对话" id="menu.modelChat" />
    </Button>
  );
};

const ConsoleSwitcher = ({
  avatarChildren,
  isAdmin,
}: {
  avatarChildren: ReactNode;
  isAdmin: boolean;
}) => {
  const location = useLocation();
  const inSystemConsole = location.pathname.startsWith('/system');

  return (
    <Dropdown
      menu={{
        items: [
          ...(isAdmin
            ? [
                {
                  icon: inSystemConsole ? (
                    <UserOutlined />
                  ) : (
                    <ControlOutlined />
                  ),
                  key: inSystemConsole ? 'personal-console' : 'system-console',
                  label: (
                    <FormattedMessage
                      defaultMessage={
                        inSystemConsole ? '个人控制台' : '系统控制台'
                      }
                      id={
                        inSystemConsole
                          ? 'header.personalConsole'
                          : 'header.systemConsole'
                      }
                    />
                  ),
                },
                { type: 'divider' as const },
              ]
            : []),
          {
            icon: <LogoutOutlined />,
            key: 'logout',
            label: (
              <FormattedMessage defaultMessage="退出登录" id="header.logout" />
            ),
          },
        ],
        onClick: ({ key }) => {
          if (key === 'system-console') {
            history.push('/system/dashboard');
            return;
          }
          if (key === 'personal-console') {
            history.push('/personal/dashboard');
            return;
          }
          clearToken();
          history.replace('/oauth/login');
        },
      }}
      placement="bottomRight"
      trigger={['click']}
    >
      {avatarChildren}
    </Dropdown>
  );
};

export async function getInitialState(): Promise<{
  settings?: Partial<LayoutSettings>;
  currentUser?: AuthedUserInfo;
}> {
  if (authenticationPaths.has(window.location.pathname)) {
    return { settings: defaultSettings as Partial<LayoutSettings> };
  }

  if (!getToken()) {
    history.replace('/oauth/login');
    return { settings: defaultSettings as Partial<LayoutSettings> };
  }

  try {
    return {
      currentUser: await getUserinfo(),
      settings: defaultSettings as Partial<LayoutSettings>,
    };
  } catch {
    clearToken();
    history.replace('/oauth/login');
    return { settings: defaultSettings as Partial<LayoutSettings> };
  }
}

export const request: RequestConfig = {
  requestInterceptors: [
    (url, options) => {
      const token = getToken();
      if (token) {
        options.headers = {
          ...options.headers,
          Authorization: `Bearer ${token.access_token}`,
        };
      }
      return { options, url };
    },
  ],
  errorConfig: {
    errorHandler: (error) => {
      if ('response' in error && error.response?.status === 401) {
        clearToken();
        if (!authenticationPaths.has(window.location.pathname)) {
          window.location.replace('/oauth/login');
        }
      }
      throw error;
    },
  },
};

export const layout: RunTimeLayoutConfig = ({ initialState }) => ({
  menuDataRender: (menuData) => {
    const inSystemConsole = history.location.pathname.startsWith('/system');
    return menuData.filter((item) =>
      inSystemConsole
        ? item.path === '/system'
        : item.path?.startsWith('/personal/'),
    );
  },
  menuItemRender: (item, dom) =>
    item.path ? (
      <Link prefetch to={item.path}>
        {dom}
      </Link>
    ) : (
      dom
    ),
  siderWidth: 220,
  menuHeaderRender: undefined,
  menuRender: (_, defaultDom) =>
    history.location.pathname === '/personal/chat' ? null : defaultDom,
  headerContentRender: () => <HeaderChatEntry />,
  actionsRender: () => [<LangDropdown key="language" />],
  avatarProps: {
    src: initialState?.currentUser?.avatar,
    title: initialState?.currentUser?.username,
    render: (_: unknown, avatarChildren: ReactNode) => {
      const isAdmin = initialState?.currentUser?.role === 'admin';
      return (
        <ConsoleSwitcher avatarChildren={avatarChildren} isAdmin={isAdmin} />
      );
    },
  },
  onPageChange: () => {
    if (!getToken() && !authenticationPaths.has(window.location.pathname)) {
      history.replace('/oauth/login');
    }
  },
  ...initialState?.settings,
});
