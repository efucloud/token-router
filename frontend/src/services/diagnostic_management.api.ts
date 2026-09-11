import { request } from '@umijs/max';

import { ModelDiagnosticInput, ModelDiagnosticResult } from './control_plane.d';

//运行管理员模型演练诊断
//
//请求方法: POST
//请求地址: /api/v1/model-diagnostics
export async function runModelDiagnostic(  data: ModelDiagnosticInput,   options?: { [key: string]: any }) {
  return request<ModelDiagnosticResult>(`/api/v1/model-diagnostics`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data,
    ...(options || {}),
  });
}
