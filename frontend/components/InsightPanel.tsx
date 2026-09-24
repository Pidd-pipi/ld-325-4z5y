'use client';

import { useEffect, useMemo, useState } from 'react';
import { BellRing, CheckCircle2, Download, Info, PackageOpen } from 'lucide-react';

import type { Product, Trend } from '@/lib/types';
import { money } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PriceTrend } from './PriceTrend';

type TrendRange = '30d' | '90d' | '1y';

interface InsightPanelProps {
  product: Product | null;
  trend: Trend | null;
  range: TrendRange;
  onRange: (range: TrendRange) => void;
  onAlert: (id: number, target: number, dropPercent: number) => Promise<void>;
}

const rangeOptions: Array<{ value: TrendRange; label: string }> = [
  { value: '30d', label: '30 天' },
  { value: '90d', label: '90 天' },
  { value: '1y', label: '1 年' },
];

// A quote counts toward the alert baseline only when the shop is approved and
// the goods are in stock — the same rule applied on the server.
function validOffers(product: Product) {
  return (product.Offers || []).filter((offer) => offer.StockStatus === 'in_stock' && offer.Supplier?.Status === 'approved');
}

export function InsightPanel({ product, trend, range, onRange, onAlert }: InsightPanelProps) {
  const [alerted, setAlerted] = useState(false);
  const [pending, setPending] = useState(false);
  useEffect(() => setAlerted(false), [product?.ID]);

  const offers = useMemo(() => (product ? validOffers(product) : []), [product]);
  const allOffers = product?.Offers || [];
  const low = offers.length ? Math.min(...offers.map((offer) => offer.UnitPrice)) : 0;
  const target = low ? Math.floor(low * 0.95) : 0;

  if (!product) {
    return <section className="insights"><p className="eyebrow">PRICE PULSE</p><h2>选择一款材料，查看它的价格脉搏。</h2><p>趋势、供应商、最低价和预警都将在这里展开。</p></section>;
  }

  const submit = async () => {
    setPending(true);
    try {
      await onAlert(product.ID, target, 5);
      setAlerted(true);
    } finally {
      setPending(false);
    }
  };

  return (
    <section className="insights">
      <div className="insight-head">
        <div>
          <p className="eyebrow">PRICE PULSE · {rangeOptions.find((option) => option.value === range)?.label}</p>
          <h2>{product.Name}</h2>
          <p>{product.Brand} · {product.Model} · 预警基线取已审核店铺的有货最低报价</p>
        </div>
        {low > 0 ? (
          <Button onClick={submit} disabled={pending || alerted} className={alerted ? 'selected' : ''}>
            {alerted ? <CheckCircle2 size={16} /> : <BellRing size={16} />} {alerted ? '预警已订阅' : `低于 ${money(target)}（再降 5%）时提醒`}
          </Button>
        ) : (
          <Button disabled><PackageOpen size={16} /> 暂无可比较的有效报价</Button>
        )}
      </div>
      {low === 0 && <p className="insight-note">该商品暂时没有“店铺已审核且有货”的有效报价，暂不可比较，也无法建立预警基线；恢复有效报价后即可订阅。</p>}
      <div className="range-tabs" role="group" aria-label="价格趋势时间范围">
        {rangeOptions.map((option) => <button key={option.value} className={range === option.value ? 'active' : ''} onClick={() => onRange(option.value)}>{option.label}</button>)}
      </div>
      <div className="trend-layout">
        <PriceTrend trend={trend} />
        <dl>
          <div><dt>{rangeOptions.find((option) => option.value === range)?.label}最低</dt><dd>{money(trend?.lowest || low)}</dd></div>
          <div><dt>{rangeOptions.find((option) => option.value === range)?.label}均价</dt><dd>{money(trend?.average || low)}</dd></div>
          <div><dt>在售有效商家</dt><dd>{offers.length} 家</dd></div>
        </dl>
      </div>
      <div className="offer-table">
        <div className="offer-title"><b>商家报价</b><span><Info size={14} /> 仅已审核 + 有货报价参与预警与最低价比较</span></div>
        {allOffers.map((offer) => {
          const isValid = offer.StockStatus === 'in_stock' && offer.Supplier?.Status === 'approved';
          const isLowest = isValid && offer.UnitPrice === low;
          return (
            <div className={isLowest ? 'offer lowest' : 'offer'} key={offer.ID}>
              <b>{offer.Supplier.Name}</b>
              <span>{offer.DeliveryDays} 天交货 · 起订 {offer.MOQ} {product.Unit}</span>
              <span>
                {isValid ? '有货' : offer.StockStatus === 'in_stock' ? '店铺待审核' : offer.StockStatus === 'out_of_stock' ? '缺货' : '停产'}
                {!isValid && <Badge tone="neutral">不参与比较</Badge>}
              </span>
              <strong>{money(offer.UnitPrice)}</strong>
            </div>
          );
        })}
      </div>
      <button className="export"><Download size={15} />导出该材料报价单</button>
    </section>
  );
}
