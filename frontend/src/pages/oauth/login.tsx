import { useEffect, useRef, useState } from 'react';
import { useIntl } from '@umijs/max';
import type { OidcConfig } from '@/services/common.d';
import { getAuthorizeInfo } from '@/services/oauth.api';
import { createOidcState, getToken } from '@/utils/auth';

const LoginPage = () => {
  const intl = useIntl();
  const [error, setError] = useState('');
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;

    if (getToken()) {
      window.location.replace('/');
      return;
    }

    const redirectToProvider = async () => {
      try {
        const config = (await getAuthorizeInfo()) as OidcConfig;
        const authorizeUrl = new URL(
          `${config.issuer.replace(/\/$/, '')}/oauth/authorize`,
        );
        authorizeUrl.searchParams.set(
          'redirect_uri',
          `${window.location.origin}/oauth/callback`,
        );
        authorizeUrl.searchParams.set('client_id', config.clientId);
        authorizeUrl.searchParams.set('response_type', 'code');
        authorizeUrl.searchParams.set('scope', 'openid profile email');
        authorizeUrl.searchParams.set('state', createOidcState());
        window.location.replace(authorizeUrl.toString());
      } catch {
        setError(intl.formatMessage({ id: 'auth.connectFailed' }));
      }
    };

    void redirectToProvider();
  }, [intl.formatMessage]);

  return (
    <main className="auth-status">
      {error || intl.formatMessage({ id: 'auth.redirecting' })}
    </main>
  );
};

export default LoginPage;
