import { request } from '@umijs/max';


//生成数据库ID
//生成数据库ID
//请求方法: GET
//请求地址: /api/v1/generateDatabaseId
export async function generateDatabaseId(  options?: { [key: string]: any }) {
  return request(`/api/v1/generateDatabaseId`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
//健康检查
//健康检查
//请求方法: GET
//请求地址: /api/v1/health
export async function health(  options?: { [key: string]: any }) {
  return request(`/api/v1/health`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
//查看应用信息
//查看应用的编译信息
//请求方法: GET
//请求地址: /api/v1/info
export async function info(  options?: { [key: string]: any }) {
  return request(`/api/v1/info`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
//存活探针
//
//请求方法: GET
//请求地址: /livez
export async function getliveness(  options?: { [key: string]: any }) {
  return request(`/livez`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
//就绪探针
//
//请求方法: GET
//请求地址: /readyz
export async function getreadiness(  options?: { [key: string]: any }) {
  return request(`/readyz`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
