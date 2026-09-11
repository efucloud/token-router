import type { AuthedUserInfo } from '@/services/common.d';

export default function access(initialState?: {
  currentUser?: AuthedUserInfo;
}) {
  const isSystemAdministrator = initialState?.currentUser?.role === 'admin';
  return {
    canUsePersonalConsole: Boolean(initialState?.currentUser),
    canUseSystemConsole: isSystemAdministrator,
  };
}
