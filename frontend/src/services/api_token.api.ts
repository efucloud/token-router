import { request } from '@umijs/max';

import { APITokenCreate, APITokenCreated, APITokenDetail, APITokenUpdate } from './api_token.d';

//删除我的 API Key
//
//请求方法: DELETE
//请求地址: /api/v1/tokens/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: API Key ID
export async function deleteMyAPIToken(
  params: {
    id: string;// API Key ID
  },
  options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request(`/api/v1/tokens/${id}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
//获取我的 API Key
//
//请求方法: GET
//请求地址: /api/v1/tokens
export async function listMyAPITokens(  options?: { [key: string]: any }) {
  return request(`/api/v1/tokens`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
//创建 API Key
//明文 API Key 仅在本响应中返回一次
//请求方法: POST
//请求地址: /api/v1/tokens
export async function createMyAPIToken(  data: APITokenCreate,   options?: { [key: string]: any }) {
  return request<APITokenCreated>(`/api/v1/tokens`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//更新我的 API Key
//
//请求方法: PUT
//请求地址: /api/v1/tokens/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: API Key ID
export async function updateMyAPIToken(
  params: {
    id: string;// API Key ID
  },
  data: APITokenUpdate,   options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<APITokenDetail>(`/api/v1/tokens/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    params: { ...rest },
    ...(options || {}),
  });
}
