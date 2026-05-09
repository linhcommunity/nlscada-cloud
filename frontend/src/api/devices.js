import apiClient from './client';

export const getDevices = (siteId) =>
  apiClient.get(`/sites/${siteId}/devices`);

export const getDevice = (siteId, deviceId) =>
  apiClient.get(`/sites/${siteId}/devices/${deviceId}`);

export const addDevice = (siteId, data) =>
  apiClient.post(`/sites/${siteId}/devices`, data);

export const deleteDevice = (siteId, deviceId) =>
  apiClient.delete(`/sites/${siteId}/devices/${deviceId}`);

export const getTags = (siteId, deviceId) =>
  apiClient.get(`/sites/${siteId}/devices/${deviceId}/tags`);

export const addTag = (siteId, deviceId, data) =>
  apiClient.post(`/sites/${siteId}/devices/${deviceId}/tags`, data);

export const deleteTag = (siteId, deviceId, tagId) =>
  apiClient.delete(`/sites/${siteId}/devices/${deviceId}/tags/${tagId}`);