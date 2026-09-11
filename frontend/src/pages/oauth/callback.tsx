import { useEffect, useRef, useState } from 'react';
import { useIntl } from '@umijs/max';
import type { AccessTokenResponse } from '@/services/common.d';
import { loginByOidc } from '@/services/oauth.api';
import { consumeOidcState, saveToken } from '@/utils/auth';

const CallbackPage = () => {
  const intl = useIntl();
  const [error, setError] = useState('');
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;

    const finishLogin = async () => {
      const params = new URLSearchParams(window.location.search);
      const providerError = params.get('error');
      const code = params.get('code');
      const state = params.get('state');

      if (providerError) {
        setError(params.get('error_description') || providerError);
        return;
      }
      if (!code || !consumeOidcState(state)) {
        setError(intl.formatMessage({ id: 'auth.invalidCallback' }));
        return;
      }

      try {
        const token = (await loginByOidc({
          code,
          redirectUri: `${window.location.origin}/oauth/callback`,
        })) as AccessTokenResponse;
        saveToken(token);
        window.location.replace('/');
      } catch {
        setError(intl.formatMessage({ id: 'auth.loginFailed' }));
      }
    };

    void finishLogin();
  }, [intl.formatMessage]);

  return (
    <main className="auth-status">
      {error || intl.formatMessage({ id: 'auth.completing' })}
    </main>
  );
};

export default CallbackPage;
