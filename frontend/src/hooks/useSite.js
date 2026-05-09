import { useAuth } from './useAuth';
import { useParams } from 'react-router-dom';

export function useSite() {
  const { currentSite, sites } = useAuth();
  const { siteId } = useParams();
  if (siteId) {
    return sites.find(s => s.id === siteId) || null;
  }
  return currentSite;
}