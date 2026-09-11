export type APITokenCreate = {
  name: string;
  allowedModels?: string[];
  tokenLimit: number;
  requestLimit: number;
  ipAllowlist?: string[];
  rpm: number;
  tpm: number;
  maxConcurrency: number;
  expiresAt?: string;
};
export type APITokenCreated = {
  token?: string;
} & APITokenDetail;
export type APITokenDetail = {
  id: string;
  name?: string;
  keyPrefix?: string;
  status?: string;
  allowedModels?: string[];
  tokenLimit?: number;
  requestLimit?: number;
  ipAllowlist?: string[];
  rpm?: number;
  tpm?: number;
  maxConcurrency?: number;
  usedTokens?: number;
  usedRequests?: number;
  expiresAt?: string;
  createdAt: string;
  updatedAt: string;
};
export type APITokenList = {
  data?: APITokenDetail[];
  total?: number;
};
export type APITokenUpdate = {
  name: string;
  allowedModels?: string[];
  tokenLimit: number;
  requestLimit: number;
  ipAllowlist?: string[];
  rpm: number;
  tpm: number;
  maxConcurrency: number;
  expiresAt?: string;
  status: string;
};
