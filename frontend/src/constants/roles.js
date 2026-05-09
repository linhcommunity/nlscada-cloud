export const ROLES = {
  ADMIN: 'admin',
  OPERATOR: 'operator',
  VIEWER: 'viewer',
  MANAGER: 'manager',
};

export const PERMISSIONS = {
  CREATE_SITE: ['admin', 'operator', 'manager'], // tất cả đều có thể tạo site
  DELETE_SITE: ['admin'],
  MANAGE_MEMBERS: ['admin'],
  ADD_DEVICE: ['admin', 'operator'],
  DELETE_DEVICE: ['admin'],
  ADD_TAG: ['admin', 'operator'],
  DELETE_TAG: ['admin'],
  VIEW_DEVICE: ['admin', 'operator', 'manager', 'viewer'],
};