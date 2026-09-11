import { request } from '@umijs/max';

import { RouteAttemptList, UsageLogList } from './control_plane.d';

//查询全部路由尝试
//
//请求方法: GET
//请求地址: /api/v1/route-attempts
//参数名: channelId 参数类型: string 参数位置: query 是否必须: false  参数说明: 渠道ID
//参数名: current 参数类型: number 参数位置: query 是否必须: false  参数说明: 页码
//参数名: pageSize 参数类型: number 参数位置: query 是否必须: false  参数说明: 每页大小
//参数名: requestId 参数类型: string 参数位置: query 是否必须: false  参数说明: 请求ID
export async function listRouteAttempts(
  params: {
    channelId?: string;// 渠道ID
    current?: number;// 页码
    pageSize?: number;// 每页大小
    requestId?: string;// 请求ID
  },
  options?: { [key: string]: any }) {
  return request<RouteAttemptList>(`/api/v1/route-attempts`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//查询用量日志
//
//请求方法: GET
//请求地址: /api/v1/usage
//参数名: accountId 参数类型: string 参数位置: query 是否必须: false  参数说明: 用户ID
//参数名: channelId 参数类型: string 参数位置: query 是否必须: false  参数说明: 渠道ID
//参数名: current 参数类型: number 参数位置: query 是否必须: false  参数说明: 页码
//参数名: end 参数类型: string 参数位置: query 是否必须: false  参数说明: 结束时间，RFC3339
//参数名: modelId 参数类型: string 参数位置: query 是否必须: false  参数说明: 模型ID
//参数名: pageSize 参数类型: number 参数位置: query 是否必须: false  参数说明: 每页大小
//参数名: requestId 参数类型: string 参数位置: query 是否必须: false  参数说明: 请求ID
//参数名: start 参数类型: string 参数位置: query 是否必须: false  参数说明: 开始时间，RFC3339
//参数名: status 参数类型: string 参数位置: query 是否必须: false  参数说明: 状态
export async function listUsage(
  params: {
    accountId?: string;// 用户ID
    channelId?: string;// 渠道ID
    current?: number;// 页码
    end?: string;// 结束时间，RFC3339
    modelId?: string;// 模型ID
    pageSize?: number;// 每页大小
    requestId?: string;// 请求ID
    start?: string;// 开始时间，RFC3339
    status?: string;// 状态
  },
  options?: { [key: string]: any }) {
  return request<UsageLogList>(`/api/v1/usage`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//查询请求的路由尝试
//
//请求方法: GET
//请求地址: /api/v1/usage/{requestId}/attempts
//参数名: requestId 参数类型: string 参数位置: path 是否必须: true  参数说明: 请求ID
export async function listUsageAttempts(
  params: {
    requestId: string;// 请求ID
  },
  options?: { [key: string]: any }) {
  const { requestId, ...rest } = params;
  return request<RouteAttemptList>(`/api/v1/usage/${requestId}/attempts`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
