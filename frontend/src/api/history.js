import apiClient from './client';

export const getDeviceData = (siteId, deviceId, params) =>
  apiClient.get(`/sites/${siteId}/devices/${deviceId}/data`, { params });