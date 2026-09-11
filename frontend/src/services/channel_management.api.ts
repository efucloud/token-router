import { request } from '@umijs/max';

import { ChannelDetail, ChannelInput, ChannelList, ChannelModelImportInput, ChannelModelImportResult, ChannelModelSyncInput, ChannelModelSyncResult, ChannelTestResult } from './control_plane.d';

//删除渠道
//
//请求方法: DELETE
//请求地址: /api/v1/channels/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 渠道ID
export async function deleteChannel(
  params: {
    id: string;// 渠道ID
  },
  options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request(`/api/v1/channels/${id}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
//查询渠道
//
//请求方法: GET
//请求地址: /api/v1/channels
//参数名: providerId 参数类型: string 参数位置: query 是否必须: false  参数说明: 供应商ID
//参数名: search 参数类型: string 参数位置: query 是否必须: false  参数说明: 名称、地址或供应商搜索
//参数名: status 参数类型: string 参数位置: query 是否必须: false  参数说明: 状态
export async function listChannels(
  params: {
    providerId?: string;// 供应商ID
    search?: string;// 名称、地址或供应商搜索
    status?: string;// 状态
  },
  options?: { [key: string]: any }) {
  return request<ChannelList>(`/api/v1/channels`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//测试渠道连接并发现模型
//
//请求方法: GET
//请求地址: /api/v1/channels/{id}/test
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 渠道ID
export async function testChannel(
  params: {
    id: string;// 渠道ID
  },
  options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<ChannelTestResult>(`/api/v1/channels/${id}/test`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
//创建渠道
//
//请求方法: POST
//请求地址: /api/v1/channels
export async function createChannel(  data: ChannelInput,   options?: { [key: string]: any }) {
  return request<ChannelDetail>(`/api/v1/channels`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//导入渠道发现的模型和路由
//
//请求方法: POST
//请求地址: /api/v1/channels/{id}/models/import
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 渠道ID
export async function importChannelModels(
  params: {
    id: string;// 渠道ID
  },
  data: ChannelModelImportInput,   options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<ChannelModelImportResult>(`/api/v1/channels/${id}/models/import`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    params: { ...rest },
    ...(options || {}),
  });
}
//同步渠道模型并清理失效模型
//
//请求方法: POST
//请求地址: /api/v1/channels/{id}/models/sync
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 渠道ID
export async function syncChannelModels(
  params: {
    id: string;// 渠道ID
  },
  data: ChannelModelSyncInput,   options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<ChannelModelSyncResult>(`/api/v1/channels/${id}/models/sync`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    params: { ...rest },
    ...(options || {}),
  });
}
//更新渠道
//
//请求方法: PUT
//请求地址: /api/v1/channels/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 渠道ID
export async function updateChannel(
  params: {
    id: string;// 渠道ID
  },
  data: ChannelInput,   options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<ChannelDetail>(`/api/v1/channels/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    params: { ...rest },
    ...(options || {}),
  });
}
