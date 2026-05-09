import apiClient from './client';

export const getSites = () => apiClient.get('/sites');

export const createSite = (name) => apiClient.post('/sites', { name });

export const updateSite = (id, data) => apiClient.put(`/sites/${id}`, data);

export const deleteSite = (id) => apiClient.delete(`/sites/${id}`);

export const getMembers = (siteId) => apiClient.get(`/sites/${siteId}/members`);

export const inviteMember = (siteId, email, role) =>
  apiClient.post(`/sites/${siteId}/members`, { email, role });

export const updateMemberRole = (siteId, memberId, role) =>
  apiClient.put(`/sites/${siteId}/members/${memberId}`, { role });

export const removeMember = (siteId, memberId) =>
  apiClient.delete(`/sites/${siteId}/members/${memberId}`);