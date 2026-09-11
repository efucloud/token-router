import { request } from '@umijs/max';

import { AIModelDetail, AIModelInput, AIModelList } from './control_plane.d';

//删除统一模型
//
//请求方法: DELETE
//请求地址: /api/v1/models/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 模型ID
export async function deleteModel(
  params: {
    id: string;// 模型ID
  },
  options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request(`/api/v1/models/${id}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
//查询统一模型
//
//请求方法: GET
//请求地址: /api/v1/models
//参数名: channelId 参数类型: string 参数位置: query 是否必须: false  参数说明: 按渠道筛选
//参数名: search 参数类型: string 参数位置: query 是否必须: false  参数说明: 名称搜索
//参数名: status 参数类型: string 参数位置: query 是否必须: false  参数说明: 状态
export async function listModels(
  params: {
    channelId?: string;// 按渠道筛选
    search?: string;// 名称搜索
    status?: string;// 状态
  },
  options?: { [key: string]: any }) {
  return request<AIModelList>(`/api/v1/models`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//创建统一模型
//
//请求方法: POST
//请求地址: /api/v1/models
export async function createModel(  data: AIModelInput,   options?: { [key: string]: any }) {
  return request<AIModelDetail>(`/api/v1/models`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//更新统一模型
//
//请求方法: PUT
//请求地址: /api/v1/models/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 模型ID
export async function updateModel(
  params: {
    id: string;// 模型ID
  },
  data: AIModelInput,   options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<AIModelDetail>(`/api/v1/models/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    params: { ...rest },
    ...(options || {}),
  });
}
