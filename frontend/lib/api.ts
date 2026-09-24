import type { ApiEnvelope, AlertGroups, OfferSubmission, OfferSubmissionResult, Product, Trend } from './types';

const API_ROOT = '/api/v1';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_ROOT}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init,
  });
  const payload = await response.json() as ApiEnvelope<T>;
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message);
  }
  return payload.data;
}

// Supplier demos run under a short-lived JWT so the repricing endpoint keeps
// its supplier RBAC while remaining exercisable from the browser.
async function supplierRequest<T>(init: RequestInit): Promise<T> {
  const auth = await request<{ token: string }>('/demo/supplier-token');
  return request<T>('/supplier/offers', {
    ...init,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${auth.token}` },
  });
}

export const api = {
  listProducts: (q = '') => request<{ items: Product[]; total: number }>(`/products?sort=rating&q=${encodeURIComponent(q)}`),
  listSuppliers: () => request<import('./types').Supplier[]>('/suppliers'),
  trend: (id: number, range = '30d') => request<Trend>(`/products/${id}/trend?range=${range}`),
  compare: (ids: number[]) => request<Product[]>('/products/compare', { method: 'POST', body: JSON.stringify({ ids }) }),
  favorite: (productId: number) => request('/favorites', { method: 'POST', body: JSON.stringify({ product_id: productId, folder: '本周采购' }) }),
  alerts: () => request<AlertGroups>('/alerts'),
  alert: (productId: number, target: number, dropPercent: number) => request('/alerts', {
    method: 'POST',
    body: JSON.stringify({ product_id: productId, target_price: target, drop_percent: dropPercent }),
  }),
  submitOffer: (input: OfferSubmission) => supplierRequest<OfferSubmissionResult>({ body: JSON.stringify(input) }),
  budget: (room: string, area: number) => request<{ Estimate: number; Payload: string }>('/budgets', { method: 'POST', body: JSON.stringify({ room_type: room, area }) }),
};
