export type AdminDashboard = {
  range?: DashboardRange;
  generatedAt?: string;
  summary?: DashboardSummary;
  previousSummary?: DashboardSummary;
  principals?: DashboardPrincipals;
  resourceHealth?: DashboardResourceHealth;
  routing?: DashboardRouting;
  trend?: DashboardTrendPoint[];
  modelUsage?: DashboardDimensionUsage[];
  userUsage?: DashboardDimensionUsage[];
  channelUsage?: DashboardDimensionUsage[];
  quotaRisks?: DashboardAlert[];
  recentFailures?: DashboardRecentRequest[];
};
export type DashboardAlert = {
  level?: string;
  code?: string;
  title?: string;
  message?: string;
};
export type DashboardDimensionUsage = {
  id: string;
  name?: string;
  requests?: number;
  totalTokens?: number;
  successRate?: number;
  p95DurationMs?: number;
  usageRatio?: number;
  status?: string;
};
export type DashboardErrorBreakdown = {
  code?: string;
  count?: number;
};
export type DashboardPrincipals = {
  activeUsers?: number;
  activeTokens?: number;
  usersNearQuota?: number;
  tokensNearQuota?: number;
  tokensExpiringSoon?: number;
};
export type DashboardQuota = {
  tokenLimit?: number;
  usedTokens?: number;
  reservedTokens?: number;
  remainingTokens?: number;
  tokenUsageRatio?: number;
  requestLimit?: number;
  usedRequests?: number;
  reservedRequests?: number;
  remainingRequests?: number;
  requestUsageRatio?: number;
};
export type DashboardRange = {
  start?: string;
  end?: string;
  timezone?: string;
  granularity?: string;
};
export type DashboardRecentRequest = {
  requestId?: string;
  createdAt: string;
  tokenPrefix?: string;
  modelName?: string;
  status?: string;
  httpStatus?: number;
  totalTokens?: number;
  durationMs?: number;
  errorCode?: string;
  attemptCount?: number;
};
export type DashboardResourceHealth = {
  activeModels?: number;
  healthyChannels?: number;
  unknownChannels?: number;
  cooldownChannels?: number;
  disabledChannels?: number;
  misconfiguredChannels?: number;
};
export type DashboardRouting = {
  upstreamAttempts?: number;
  failoverRequests?: number;
  successfulFailovers?: number;
  failoverSuccessRate?: number;
};
export type DashboardSummary = {
  totalRequests?: number;
  succeededRequests?: number;
  failedRequests?: number;
  rejectedRequests?: number;
  successRate?: number;
  promptTokens?: number;
  completionTokens?: number;
  totalTokens?: number;
  estimatedRequests?: number;
  estimatedRatio?: number;
  averageDurationMs?: number;
  p95DurationMs?: number;
};
export type DashboardTokenStatus = {
  active?: number;
  disabled?: number;
  expired?: number;
  expiringSoon?: number;
};
export type DashboardTrendPoint = {
  bucket?: string;
  requests?: number;
  succeeded?: number;
  failed?: number;
  rejected?: number;
  promptTokens?: number;
  completionTokens?: number;
  totalTokens?: number;
  averageDurationMs?: number;
  failoverRequests?: number;
};
export type MyDashboard = {
  range?: DashboardRange;
  generatedAt?: string;
  quota?: DashboardQuota;
  tokenStatus?: DashboardTokenStatus;
  summary?: DashboardSummary;
  previousSummary?: DashboardSummary;
  trend?: DashboardTrendPoint[];
  modelUsage?: DashboardDimensionUsage[];
  errorBreakdown?: DashboardErrorBreakdown[];
  recentRequests?: DashboardRecentRequest[];
  alerts?: DashboardAlert[];
};
