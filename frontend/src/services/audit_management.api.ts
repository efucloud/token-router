import { request } from '@umijs/max';

import { AuditLogList } from './control_plane.d';

//查询管理操作审计日志
//
//请求方法: GET
//请求地址: /api/v1/audit-logs
//参数名: action 参数类型: string 参数位置: query 是否必须: false  参数说明: 操作
//参数名: current 参数类型: number 参数位置: query 是否必须: false  参数说明: 页码
//参数名: operatorId 参数类型: string 参数位置: query 是否必须: false  参数说明: 操作者ID
//参数名: pageSize 参数类型: number 参数位置: query 是否必须: false  参数说明: 每页大小
//参数名: resourceType 参数类型: string 参数位置: query 是否必须: false  参数说明: 资源类型
//参数名: result 参数类型: string 参数位置: query 是否必须: false  参数说明: 执行结果
export async function listAuditLogs(
  params: {
    action?: string;// 操作
    current?: number;// 页码
    operatorId?: string;// 操作者ID
    pageSize?: number;// 每页大小
    resourceType?: string;// 资源类型
    result?: string;// 执行结果
  },
  options?: { [key: string]: any }) {
  return request<AuditLogList>(`/api/v1/audit-logs`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    params: params,
    ...(options || {}),
  });
}
