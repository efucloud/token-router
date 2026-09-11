import type { AccessTokenResponse } from '@/services/common.d';

const tokenKey = 'token-router.oidc-token';
const stateKey = 'token-router.oidc-state';

export const clearToken = () => {
  localStorage.removeItem(tokenKey);
};

export const getToken = (): AccessTokenResponse | undefined => {
  try {
    const value = localStorage.getItem(tokenKey);
    if (!value) return undefined;

    const token = JSON.parse(value) as AccessTokenResponse;
    const expiresAt = token.expires_in * 1000;
    if (
      !token.access_token ||
      (expiresAt && expiresAt <= Date.now() + 30_000)
    ) {
      clearToken();
      return undefined;
    }
    return token;
  } catch {
    clearToken();
    return undefined;
  }
};

export const saveToken = (token: AccessTokenResponse) => {
  localStorage.setItem(tokenKey, JSON.stringify(token));
};

export const createOidcState = (): string => {
  const bytes = crypto.getRandomValues(new Uint8Array(32));
  const state = Array.from(bytes, (byte) =>
    byte.toString(16).padStart(2, '0'),
  ).join('');
  sessionStorage.setItem(stateKey, state);
  return state;
};

export const consumeOidcState = (receivedState: string | null): boolean => {
  const expectedState = sessionStorage.getItem(stateKey);
  sessionStorage.removeItem(stateKey);
  return Boolean(
    receivedState && expectedState && receivedState === expectedState,
  );
};
