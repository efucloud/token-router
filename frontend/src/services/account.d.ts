// AccountCreate 账户信息创建
// 未来账户信息修改只能从eauth中
export type AccountCreate = {
  //记录ID
  //最大长度: 50
  id: string;
  //用户名
  //最大长度: 255
  username: string;
  //昵称，如中文名
  //最大长度: 255
  nickname: string;
  //组织角色
  //默认值: none
  //最大长度: 255
  role: string;
  //是否有效
  //默认值: true
  enable?: boolean;
  //邮箱
  //最大长度: 255
  email: string;
  //手机号码
  //最大长度: 255
  phone: string;
  //默认语言
  //默认值: zh
  //最大长度: 255
  language: string;
};
// AccountDetail 账户详情
export type AccountDetail = {
  //主键
  //最大长度: 50
  id: string;
  //创建时间
  createdAt: string;
  //更新时间
  updatedAt: string;
  //创建者
  //最大长度: 50
  creatorId: string;
  //更新者
  //最大长度: 50
  updaterId: string;
  //软删除
  deletedAt?: string;
  //用户名
  //最大长度: 255
  username: string;
  //昵称，如中文名
  //最大长度: 255
  nickname: string;
  //工号
  //最大长度: 255
  jobNumber?: string;
  //系统角色
  //默认值: none
  //最大长度: 255
  role: string;
  //是否有效
  //默认值: true
  enable?: boolean;
  //邮箱
  //最大长度: 255
  email: string;
  //手机号码
  //最大长度: 255
  phone: string;
  //默认语言
  //最大长度: 255
  language: string;
  //头像
  //最大长度: 1000
  avatar?: string;
  //Token 总量限额，0 表示不限制
  tokenLimit?: number;
  //成功请求次数限额，0 表示不限制
  requestLimit?: number;
  //已使用 Token 数
  usedTokens?: number;
  //已使用成功请求数
  usedRequests?: number;
};
// AccountDetailList  账户列表响应
export type AccountDetailList = {
  //当前页数据
  data?: AccountDetail[];
  //数据库满足条件的数据总数
  total: number;
};
// AccountQuotaUpdate 更新用户聚合限额；0 表示不限制。
export type AccountQuotaUpdate = {
  tokenLimit: number;
  requestLimit: number;
};
// AccountResetPassword 账户修改密码
export type AccountResetPassword = {
  //主键
  //最大长度: 50
  id: string;
  //密码
  newPassword: string;
};
// AccountRole 账户系统角色设置
// 设置账户在系统中的角色
export type AccountRole = {
  //主键
  ids: string[];
  //角色
  //默认值: none
  //最大长度: 255
  role: string;
};
// AccountStatus 账户信息禁用/启用
// 账户禁用后，用户将不能登陆该系统
export type AccountStatus = {
  //主键
  ids: string[];
  //是否有效
  //默认值: true
  enable?: boolean;
};
// AccountUpdate 账户信息更新
// 更新账户信息，未来只能在eauth中更新
export type AccountUpdate = {
  //主键
  //最大长度: 50
  id: string;
  //用户名
  //最大长度: 255
  username: string;
  //昵称，如中文名
  //最大长度: 255
  nickname: string;
  //邮箱
  //最大长度: 255
  email: string;
  //手机号码
  //最大长度: 255
  phone: string;
  //默认语言
  //默认值: zh
  //最大长度: 255
  language: string;
};
// AccountQuotaUpdate 更新用户聚合限额；0 表示不限制。
export type SimpleAccountDetail = {
  //主键
  //最大长度: 50
  id: string;
  //用户名
  //最大长度: 255
  username: string;
  //昵称，如中文名
  //最大长度: 255
  nickname: string;
  //工号
  //最大长度: 255
  jobNumber?: string;
  //系统角色
  //默认值: none
  //最大长度: 255
  role: string;
  //是否有效
  //默认值: true
  enable?: boolean;
  //账户类型，企业用户时role才可以不为none
  //最大长度: 255
  category: string;
  //邮箱
  //最大长度: 255
  email: string;
  //手机号码
  //最大长度: 255
  phone: string;
  //默认语言
  //最大长度: 255
  language: string;
  //头像
  //最大长度: 1000
  avatar?: string;
  //Token 总量限额，0 表示不限制
  tokenLimit?: number;
  //成功请求次数限额，0 表示不限制
  requestLimit?: number;
  //已使用 Token 数
  usedTokens?: number;
  //已使用成功请求数
  usedRequests?: number;
};
