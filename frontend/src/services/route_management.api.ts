import { request } from '@umijs/max';

import { ModelRouteDetail, ModelRouteInput, ModelRouteList } from './control_plane.d';

//删除模型路由
//
//请求方法: DELETE
//请求地址: /api/v1/routes/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 路由ID
export async function deleteRoute(
  params: {
    id: string;// 路由ID
  },
  options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request(`/api/v1/routes/${id}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
//查询模型路由
//
//请求方法: GET
//请求地址: /api/v1/routes
//参数名: channelId 参数类型: string 参数位置: query 是否必须: false  参数说明: 渠道ID
//参数名: modelId 参数类型: string 参数位置: query 是否必须: false  参数说明: 统一模型ID
//参数名: status 参数类型: string 参数位置: query 是否必须: false  参数说明: 状态
export async function listRoutes(
  params: {
    channelId?: string;// 渠道ID
    modelId?: string;// 统一模型ID
    status?: string;// 状态
  },
  options?: { [key: string]: any }) {
  return request<ModelRouteList>(`/api/v1/routes`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//创建模型路由
//
//请求方法: POST
//请求地址: /api/v1/routes
export async function createRoute(  data: ModelRouteInput,   options?: { [key: string]: any }) {
  return request<ModelRouteDetail>(`/api/v1/routes`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//更新模型路由
//
//请求方法: PUT
//请求地址: /api/v1/routes/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 路由ID
export async function updateRoute(
  params: {
    id: string;// 路由ID
  },
  data: ModelRouteInput,   options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<ModelRouteDetail>(`/api/v1/routes/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    params: { ...rest },
    ...(options || {}),
  });
}
