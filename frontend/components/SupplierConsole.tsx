'use client';

import { useMemo, useState } from 'react';
import { Store, Truck } from 'lucide-react';

import type { Product, Supplier } from '@/lib/types';
import { money } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

interface SupplierConsoleProps {
  products: Product[];
  suppliers: Supplier[];
  busy: boolean;
  onSubmit: (input: {
    product_id: number;
    supplier_id: number;
    unit_price: number;
    moq: number;
    freight: string;
    delivery_days: number;
    stock_status: 'in_stock' | 'out_of_stock' | 'discontinued';
  }) => Promise<void>;
}

const stockOptions = [
  { value: 'in_stock', label: '有货' },
  { value: 'out_of_stock', label: '缺货' },
  { value: 'discontinued', label: '停产' },
] as const;

export function SupplierConsole({ products, suppliers, busy, onSubmit }: SupplierConsoleProps) {
  const firstProduct = products[0]?.ID ?? 0;
  const [productId, setProductId] = useState(firstProduct);
  const approvedSuppliers = useMemo(() => suppliers.filter((item) => item.Status === 'approved'), [suppliers]);
  const [supplierId, setSupplierId] = useState(approvedSuppliers[0]?.ID ?? 0);
  const [price, setPrice] = useState(300);
  const [moq, setMoq] = useState(5);
  const [freight, setFreight] = useState('包邮');
  const [deliveryDays, setDeliveryDays] = useState(3);
  const [stockStatus, setStockStatus] = useState<'in_stock' | 'out_of_stock' | 'discontinued'>('in_stock');

  const selected = products.find((item) => item.ID === productId);
  const lowest = selected && selected.Offers.length ? Math.min(...selected.Offers.map((offer) => offer.UnitPrice)) : 0;

  const submit = async () => {
    await onSubmit({
      product_id: productId,
      supplier_id: supplierId,
      unit_price: price,
      moq,
      freight,
      delivery_days: deliveryDays,
      stock_status: stockStatus,
    });
  };

  return (
    <section id="supplier" className="supplier-console">
      <div className="section-title">
        <div>
          <p className="eyebrow">SUPPLIER REPRICING · 供应商改价</p>
          <h2>提交一笔新报价，<em>预警立即重算。</em></h2>
        </div>
        <div className="catalog-tools">
          <p>演示入口：以已审核供应商身份提交报价。完全相同的报价视为同一版本重复提交，只保留一条记录且不会重复通知。</p>
        </div>
      </div>
      <div className="supplier-layout">
        <div className="calculator">
          <div className="calc-heading"><Store size={20} /><b>供应商报价录入</b></div>
          <label>建材
            <select value={productId} onChange={(event) => setProductId(Number(event.target.value))}>
              {products.map((product) => <option key={product.ID} value={product.ID}>{product.Name}（当前最低 {money(Math.min(...product.Offers.map((offer) => offer.UnitPrice)))}）</option>)}
            </select>
          </label>
          <label>店铺
            <select value={supplierId} onChange={(event) => setSupplierId(Number(event.target.value))}>
              {approvedSuppliers.map((supplier) => <option key={supplier.ID} value={supplier.ID}>{supplier.Name}（已审核）</option>)}
              {suppliers.filter((item) => item.Status !== 'approved').map((supplier) => <option key={supplier.ID} value={supplier.ID} disabled>{supplier.Name}（{supplier.Status === 'pending' ? '待审核' : '未通过'}，报价不生效）</option>)}
            </select>
          </label>
          <div className="calc-row">
            <label>单价（元）<input type="number" min="1" value={price} onChange={(event) => setPrice(Number(event.target.value))} /></label>
            <label>起订量<input type="number" min="1" value={moq} onChange={(event) => setMoq(Number(event.target.value))} /></label>
          </div>
          <div className="calc-row">
            <label>交货周期（天）<input type="number" min="1" max="90" value={deliveryDays} onChange={(event) => setDeliveryDays(Number(event.target.value))} /></label>
            <label>库存状态
              <select value={stockStatus} onChange={(event) => setStockStatus(event.target.value as typeof stockStatus)}>
                {stockOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
          </div>
          <label>运费说明<input value={freight} maxLength={60} onChange={(event) => setFreight(event.target.value)} /></label>
          <Button onClick={submit} disabled={busy || !supplierId}>{busy ? '提交中…' : '提交新报价'}</Button>
        </div>
        <aside className="supplier-hint">
          <Badge tone="neutral">当前基线参考</Badge>
          <h3>{selected?.Name}</h3>
          <p>{selected ? `${selected.Brand} · ${selected.Model}` : ''}</p>
          <div className="hint-price"><span>订阅基线 / 当前最低有效报价</span><strong>{lowest ? money(lowest) : '暂无有效报价'}</strong></div>
          <p className="hint-copy"><Truck size={14} /> 新价相对订阅基线的降幅达到订阅比例、且不高于目标价时，个人中心的订阅会变为「已触发」。</p>
        </aside>
      </div>
    </section>
  );
}
