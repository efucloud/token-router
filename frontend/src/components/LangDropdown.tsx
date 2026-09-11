import { CheckOutlined, GlobalOutlined } from '@ant-design/icons';
import { getAllLocales, getLocale, setLocale } from '@umijs/max';
import type { MenuProps } from 'antd';
import { Button, Dropdown } from 'antd';
import { useMemo } from 'react';

const localeLabels: Record<string, string> = {
  'zh-CN': '🇨🇳 简体中文',
  'en-US': '🇺🇸 English',
};

export const LangDropdown = () => {
  const locales = useMemo(() => getAllLocales(), []);
  const currentLocale = getLocale();
  const supportedLocales = locales.filter((locale) => locale in localeLabels);
  const items: MenuProps['items'] = supportedLocales.map((locale) => ({
    key: locale,
    icon:
      locale === currentLocale ? (
        <CheckOutlined style={{ color: '#52c41a' }} />
      ) : (
        <span style={{ display: 'inline-block', width: 14 }} />
      ),
    label: localeLabels[locale],
  }));

  return (
    <Dropdown
      arrow
      menu={{
        items,
        onClick: ({ key }) => setLocale(key, false),
        selectedKeys: [currentLocale],
        style: { minWidth: 180 },
      }}
      placement="bottomRight"
    >
      <Button aria-label="语言切换" icon={<GlobalOutlined />} type="text" />
    </Dropdown>
  );
};
