import { request } from '@umijs/max';

import { AdminDashboard, MyDashboard } from './dashboard.d';

//获取管理员运行看板
//返回企业网关全局用量、渠道健康、路由切换和限额风险
//请求方法: GET
//请求地址: /api/v1/dashboard/admin
//参数名: range 参数类型: string 参数位置: query 是否必须: false  参数说明: 时间范围: 24h、7d、30d
export async function getAdminDashboard(
  params: {
    range?: string;// 时间范围: 24h、7d、30d
  },
  options?: { [key: string]: any }) {
  return request<AdminDashboard>(`/api/v1/dashboard/admin`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//获取当前用户工作台
//只返回当前 OIDC 用户的限额、Token 和调用聚合数据
//请求方法: GET
//请求地址: /api/v1/dashboard/me
//参数名: range 参数类型: string 参数位置: query 是否必须: false  参数说明: 时间范围: 24h、7d、30d
export async function getMyDashboard(
  params: {
    range?: string;// 时间范围: 24h、7d、30d
  },
  options?: { [key: string]: any }) {
  return request<MyDashboard>(`/api/v1/dashboard/me`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//获取指定用户工作台
//管理员按用户查看与该用户个人工作台口径一致的限额、Token 和调用聚合数据
//请求方法: GET
//请求地址: /api/v1/dashboard/users/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 用户 ID
//参数名: range 参数类型: string 参数位置: query 是否必须: false  参数说明: 时间范围: 24h、7d、30d
export async function getUserDashboard(
  params: {
    id: string;// 用户 ID
    range?: string;// 时间范围: 24h、7d、30d
  },
  options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<MyDashboard>(`/api/v1/dashboard/users/${id}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
