import { Select } from 'antd';
import { useAuth } from '../store/AuthContext';

export default function SiteSelector() {
  const { sites, currentSite, changeSite } = useAuth();

  if (!sites || sites.length === 0) return null;

  return (
    <Select
      value={currentSite?.id}
      onChange={(siteId) => {
        const site = sites.find(s => s.id === siteId);
        if (site) changeSite(site);
      }}
      style={{ minWidth: 200 }}
      options={sites.map(s => ({ label: s.name, value: s.id }))}
    />
  );
}