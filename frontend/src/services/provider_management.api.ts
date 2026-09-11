import { request } from '@umijs/max';

import { ProviderDetail, ProviderInput, ProviderList } from './control_plane.d';

//删除供应商
//
//请求方法: DELETE
//请求地址: /api/v1/providers/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 供应商ID
export async function deleteProvider(
  params: {
    id: string;// 供应商ID
  },
  options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request(`/api/v1/providers/${id}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
//查询供应商
//
//请求方法: GET
//请求地址: /api/v1/providers
//参数名: search 参数类型: string 参数位置: query 是否必须: false  参数说明: 名称搜索
//参数名: status 参数类型: string 参数位置: query 是否必须: false  参数说明: 状态
export async function listProviders(
  params: {
    search?: string;// 名称搜索
    status?: string;// 状态
  },
  options?: { [key: string]: any }) {
  return request<ProviderList>(`/api/v1/providers`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//创建供应商
//
//请求方法: POST
//请求地址: /api/v1/providers
export async function createProvider(  data: ProviderInput,   options?: { [key: string]: any }) {
  return request<ProviderDetail>(`/api/v1/providers`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//更新供应商
//
//请求方法: PUT
//请求地址: /api/v1/providers/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 供应商ID
export async function updateProvider(
  params: {
    id: string;// 供应商ID
  },
  data: ProviderInput,   options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<ProviderDetail>(`/api/v1/providers/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    params: { ...rest },
    ...(options || {}),
  });
}
