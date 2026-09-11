import { request } from '@umijs/max';


//
//
//请求方法: GET
//请求地址: /v1/models
export async function getmodels(  options?: { [key: string]: any }) {
  return request(`/v1/models`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
//
//
//请求方法: POST
//请求地址: /v1/chat/completions
export async function postchat(  options?: { [key: string]: any }) {
  return request(`/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
//
//
//请求方法: POST
//请求地址: /v1/embeddings
export async function postembeddings(  options?: { [key: string]: any }) {
  return request(`/v1/embeddings`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
//
//
//请求方法: POST
//请求地址: /v1/responses
export async function postresponses(  options?: { [key: string]: any }) {
  return request(`/v1/responses`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    ...(options || {}),
  });
}
