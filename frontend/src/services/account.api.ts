import { request } from '@umijs/max';

import { AccountCreate, AccountDetail, AccountDetailList, AccountQuotaUpdate, AccountRole, AccountStatus, AccountUpdate } from './account.d';
import { BatchOperationIds } from './common.d';

//删除用户
//删除用户信息详情
//请求方法: DELETE
//请求地址: /api/v1/account
export async function deleteAccount(  data: BatchOperationIds,   options?: { [key: string]: any }) {
  return request(`/api/v1/account`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//获取用户列表
//获取用户列表
//请求方法: GET
//请求地址: /api/v1/account
//参数名: current 参数类型: number 参数位置: query 是否必须: false  参数说明: 页码
//参数名: email 参数类型: string 参数位置: query 是否必须: false  参数说明: 邮箱
//参数名: ids 参数类型: string 参数位置: query 是否必须: false  参数说明: 数据库记录ID数组,逗号分隔
//参数名: jobNumber 参数类型: string 参数位置: query 是否必须: false  参数说明: 工号
//参数名: nickname 参数类型: string 参数位置: query 是否必须: false  参数说明: 昵称
//参数名: order 参数类型: string 参数位置: query 是否必须: false  参数说明: 排序
//参数名: pageSize 参数类型: number 参数位置: query 是否必须: false  参数说明: 每页大小
//参数名: phone 参数类型: string 参数位置: query 是否必须: false  参数说明: 电话号码
//参数名: role 参数类型: string 参数位置: query 是否必须: false  参数说明: 系统角色
//参数名: search 参数类型: string 参数位置: query 是否必须: false  参数说明: 搜索
//参数名: username 参数类型: string 参数位置: query 是否必须: false  参数说明: 账户名英文
export async function listAccount(
  params: {
    current?: number;// 页码
    email?: string;// 邮箱
    ids?: string;// 数据库记录ID数组,逗号分隔
    jobNumber?: string;// 工号
    nickname?: string;// 昵称
    order?: string;// 排序
    pageSize?: number;// 每页大小
    phone?: string;// 电话号码
    role?: string;// 系统角色
    search?: string;// 搜索
    username?: string;// 账户名英文
  },
  options?: { [key: string]: any }) {
  return request<AccountDetailList>(`/api/v1/account`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
//获取用户详情
//获取用户信息详情
//请求方法: GET
//请求地址: /api/v1/account/{id}
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 记录ID
export async function getAccount(
  params: {
    id: string;// 记录ID
  },
  options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request<AccountDetail>(`/api/v1/account/${id}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: { ...rest },
    ...(options || {}),
  });
}
//创建用户
//创建用户信息
//请求方法: POST
//请求地址: /api/v1/account
export async function createAccount(  data: AccountCreate,   options?: { [key: string]: any }) {
  return request<AccountDetail>(`/api/v1/account`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//系统角色设置
//系统角色设置,admin: 管理员，view: 查看者， edit: 编辑者， none: 无权限，仅为系统成员
//请求方法: POST
//请求地址: /api/v1/account/role
export async function setAccountRole(  data: AccountRole,   options?: { [key: string]: any }) {
  return request<AccountDetail>(`/api/v1/account/role`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//启用禁用
//启用禁用,修改账户状态
//请求方法: POST
//请求地址: /api/v1/account/status
export async function changeAccountStatus(  data: AccountStatus,   options?: { [key: string]: any }) {
  return request(`/api/v1/account/status`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//更新用户信息
//更新用户信息
//请求方法: PUT
//请求地址: /api/v1/account
export async function updateAccount(  data: AccountUpdate,   options?: { [key: string]: any }) {
  return request<AccountDetail>(`/api/v1/account`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
//更新用户聚合限额
//只修改限额，不修改累计用量；0表示不限制
//请求方法: PUT
//请求地址: /api/v1/account/{id}/quota
//参数名: id 参数类型: string 参数位置: path 是否必须: true  参数说明: 用户ID
export async function updateAccountQuota(
  params: {
    id: string;// 用户ID
  },
  data: AccountQuotaUpdate,   options?: { [key: string]: any }) {
  const { id, ...rest } = params;
  return request(`/api/v1/account/${id}/quota`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    params: { ...rest },
    ...(options || {}),
  });
}
