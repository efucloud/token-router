import { CheckOutlined, CopyOutlined } from '@ant-design/icons';
import XMarkdown, { type ComponentProps } from '@ant-design/x-markdown';
import { useIntl } from '@umijs/max';
import { Button, message as toast } from 'antd';
import { Children, useState } from 'react';
import styles from './chat_markdown_components.less';

type ChatMarkdownProps = {
  content: string;
  streaming?: boolean;
};

const MarkdownCode = ({
  block,
  children,
  className,
  lang,
  streamStatus,
}: ComponentProps) => {
  const intl = useIntl();
  const [copied, setCopied] = useState(false);
  const code = Children.toArray(children).join('').replace(/\n$/, '');
  const language = lang?.split(/\s+/)[0] || 'code';

  if (!block) {
    return <code className={styles.inlineCode}>{children}</code>;
  }

  const copyCode = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      toast.error(intl.formatMessage({ id: 'chat.copyCodeFailed' }));
    }
  };

  return (
    <span
      className={`${styles.codeBlock} ${streamStatus === 'loading' ? styles.codeStreaming : ''}`}
    >
      <span className={styles.codeHeader}>
        <span className={styles.codeLanguage}>{language}</span>
        <Button
          aria-label={intl.formatMessage({ id: 'chat.copyCode' })}
          className={styles.copyButton}
          icon={copied ? <CheckOutlined /> : <CopyOutlined />}
          onClick={() => void copyCode()}
          size="small"
          title={intl.formatMessage({
            id: copied ? 'chat.codeCopied' : 'chat.copyCode',
          })}
          type="text"
        />
      </span>
      <code className={`${styles.codeBody} ${className || ''}`}>{code}</code>
    </span>
  );
};

const markdownComponents = { code: MarkdownCode };

export const ChatMarkdown = ({ content, streaming }: ChatMarkdownProps) => (
  <XMarkdown
    className={styles.markdown}
    components={markdownComponents}
    content={content}
    escapeRawHtml
    openLinksInNewTab
    streaming={{
      hasNextChunk: streaming,
      enableAnimation: true,
      animationConfig: { fadeDuration: 120, easing: 'ease-out' },
      tail: { content: '▋' },
    }}
  />
);
