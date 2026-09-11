export type AIModelDetail = {
  id: string;
  name?: string;
  displayName?: string;
  description?: string;
  modality?: string;
  contextWindow?: number;
  maxOutputTokens?: number;
  capabilities?: string[];
  status?: string;
  routeCount?: number;
  routeId?: string;
  routeStatus?: string;
  channelId?: string;
  channelName?: string;
  providerName?: string;
  upstreamModel?: string;
  version?: number;
  createdAt: string;
  updatedAt: string;
};
export type AIModelInput = {
  name: string;
  displayName: string;
  description: string;
  modality: string;
  contextWindow: number;
  maxOutputTokens: number;
  capabilities?: string[];
  status: string;
  version: number;
};
export type AIModelList = {
  data?: AIModelDetail[];
  total?: number;
};
export type AuditLogDetail = {
  id: string;
  operatorId?: string;
  operatorUsername?: string;
  action?: string;
  resourceType?: string;
  resourceId?: string;
  method?: string;
  routePath?: string;
  summary?: Record<string, string>;
  result?: string;
  statusCode?: number;
  requestId?: string;
  remoteIp?: string;
  createdAt: string;
};
export type AuditLogList = {
  data?: AuditLogDetail[];
  total?: number;
};
export type ChannelDetail = {
  id: string;
  providerId?: string;
  providerName?: string;
  name?: string;
  baseUrl?: string;
  credentialConfigured?: boolean;
  priority?: number;
  weight?: number;
  status?: string;
  timeoutSeconds?: number;
  maxConcurrency?: number;
  healthStatus?: string;
  cooldownUntil?: string;
  consecutiveFailures?: number;
  lastCheckedAt?: string;
  lastLatencyMs?: number;
  lastError?: string;
  routeCount?: number;
  version?: number;
  createdAt: string;
  updatedAt: string;
};
export type ChannelInput = {
  providerId: string;
  name: string;
  baseUrl: string;
  apiKey?: string;
  priority: number;
  weight: number;
  status: string;
  timeoutSeconds: number;
  maxConcurrency: number;
  version: number;
};
export type ChannelList = {
  data?: ChannelDetail[];
  total?: number;
};
export type ChannelModelImportInput = {
  models: string[];
};
export type ChannelModelImportItem = {
  modelId?: string;
  modelName?: string;
  upstreamModel?: string;
  modelCreated?: boolean;
  routeCreated?: boolean;
};
export type ChannelModelImportResult = {
  modelsCreated?: number;
  routesCreated?: number;
  skipped?: number;
  items?: ChannelModelImportItem[];
};
export type ChannelModelSyncInput = {
};
export type ChannelModelSyncResult = {
  discovered?: number;
  modelsCreated?: number;
  routesCreated?: number;
  routesDeleted?: number;
  modelsDeleted?: number;
  unchanged?: number;
};
export type ChannelTestResult = {
  success?: boolean;
  httpStatus?: number;
  latencyMs?: number;
  modelCount?: number;
  models?: string[];
  message?: string;
};
export type ModelDiagnosticAttempt = {
  sequence?: number;
  routeId?: string;
  channelId?: string;
  channelName?: string;
  providerName?: string;
  upstreamModel?: string;
  httpStatus?: number;
  durationMs?: number;
  errorCode?: string;
  errorSummary?: string;
};
export type ModelDiagnosticInput = {
  modelId: string;
  prompt: string;
  maxOutputTokens: number;
};
export type ModelDiagnosticResult = {
  success?: boolean;
  requestId?: string;
  modelId?: string;
  modelName?: string;
  outputText?: string;
  ttftMs?: number;
  durationMs?: number;
  promptTokens?: number;
  completionTokens?: number;
  totalTokens?: number;
  estimated?: boolean;
  finalChannelId?: string;
  finalChannelName?: string;
  message?: string;
  attempts?: ModelDiagnosticAttempt[];
};
export type ModelRouteDetail = {
  id: string;
  modelId?: string;
  modelName?: string;
  channelId?: string;
  channelName?: string;
  providerName?: string;
  upstreamModel?: string;
  priority?: number;
  weight?: number;
  status?: string;
  version?: number;
  createdAt: string;
  updatedAt: string;
};
export type ModelRouteInput = {
  modelId: string;
  channelId: string;
  upstreamModel: string;
  priority: number;
  weight: number;
  status: string;
  version: number;
};
export type ModelRouteList = {
  data?: ModelRouteDetail[];
  total?: number;
};
export type ProviderDetail = {
  id: string;
  name?: string;
  type?: string;
  status?: string;
  config?: Record<string, any>;
  channelCount?: number;
  version?: number;
  createdAt: string;
  updatedAt: string;
};
export type ProviderInput = {
  name: string;
  type: string;
  status: string;
  config?: Record<string, any>;
  version: number;
};
export type ProviderList = {
  data?: ProviderDetail[];
  total?: number;
};
export type RouteAttemptDetail = {
  id: string;
  requestId?: string;
  routeId?: string;
  channelId?: string;
  channelName?: string;
  sequence?: number;
  httpStatus?: number;
  durationMs?: number;
  errorCode?: string;
  errorSummary?: string;
  createdAt: string;
};
export type RouteAttemptList = {
  data?: RouteAttemptDetail[];
  total?: number;
};
export type UsageLogDetail = {
  id: string;
  requestId?: string;
  accountId?: string;
  username?: string;
  apiTokenId?: string;
  tokenPrefix?: string;
  modelId?: string;
  modelName?: string;
  channelId?: string;
  channelName?: string;
  endpoint?: string;
  streaming?: boolean;
  promptTokens?: number;
  completionTokens?: number;
  totalTokens?: number;
  estimated?: boolean;
  status?: string;
  httpStatus?: number;
  durationMs?: number;
  errorCode?: string;
  errorSummary?: string;
  createdAt: string;
};
export type UsageLogList = {
  data?: UsageLogDetail[];
  total?: number;
};
