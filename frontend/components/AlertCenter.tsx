'use client';

import { BellRing, CheckCircle2, PackageX, RefreshCw } from 'lucide-react';

import type { PriceAlert } from '@/lib/types';
import { cnDate, money } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';

interface AlertCenterProps {
  alerts: PriceAlert[];
  loading: boolean;
  onRefresh: () => void;
}

function dropOf(alert: PriceAlert): number {
  if (!alert.comparable || alert.baseline_price <= 0) return 0;
  return Math.round(((alert.baseline_price - alert.current_price) / alert.baseline_price) * 1000) / 10;
}

function AlertRow({ alert }: { alert: PriceAlert }) {
  const triggered = alert.status === 'triggered';
  const drop = dropOf(alert);
  return (
    <div className={triggered ? 'alert-row triggered' : 'alert-row'}>
      <div className="alert-product">
        <b>{alert.product.name}</b>
        <span>目标价 {money(alert.target_price)} · 降幅 ≥ {alert.drop_percent}%</span>
      </div>
      <div className="alert-price">
        <span>原价 {money(alert.baseline_price)}</span>
        {alert.comparable
          ? <strong>{money(alert.current_price)}</strong>
          : <em className="uncomparable">暂不可比较</em>}
      </div>
      <div className="alert-meta">
        {triggered
          ? <Badge tone="good"><CheckCircle2 size={11} /> 已触发</Badge>
          : <Badge tone="neutral"><BellRing size={11} /> 进行中</Badge>}
        <span>降幅 {alert.comparable ? `${drop.toFixed(1)}%` : '—'}</span>
        {triggered && alert.triggered_at && <span>触发于 {cnDate(alert.triggered_at)}</span>}
        {triggered && alert.supplier && <span>{alert.supplier}</span>}
      </div>
      <div className="alert-state">
        {triggered && (alert.purchasable
          ? <Badge tone="good">当前可采购</Badge>
          : <Badge tone="alert"><PackageX size={11} /> 暂不可采购</Badge>)}
        {!triggered && !alert.comparable && <Badge tone="alert">该商品暂时没有有效报价</Badge>}
        {!triggered && alert.comparable && <span className="watch-note">等待降价达标</span>}
      </div>
    </div>
  );
}

export function AlertCenter({ alerts, loading, onRefresh }: AlertCenterProps) {
  const active = alerts.filter((alert) => alert.status === 'active');
  const triggered = alerts.filter((alert) => alert.status === 'triggered');
  return (
    <section id="alerts" className="alerts">
      <div className="section-title">
        <div>
          <p className="eyebrow">MY PRICE WATCH</p>
          <h2>我的价格预警，<em>降价达标后只提醒一次。</em></h2>
        </div>
        <div className="catalog-tools">
          <p>基线为订阅时已审核、有货店铺的最低有效报价。触发后可继续查看当前可采购状态。</p>
          <button className="refresh-btn" onClick={onRefresh} disabled={loading}>
            <RefreshCw size={13} className={loading ? 'spin' : ''} /> {loading ? '刷新中…' : '刷新状态'}
          </button>
        </div>
      </div>
      <div className="alert-groups">
        <div className="alert-group">
          <div className="alert-group-head"><BellRing size={15} /><b>进行中</b><span>{active.length}</span></div>
          {active.length === 0
            ? <p className="alert-empty">还没有进行中的订阅，在下方材料详情里点“低于目标价提醒”即可建立。</p>
            : active.map((alert) => <AlertRow key={alert.id} alert={alert} />)}
        </div>
        <div className="alert-group">
          <div className="alert-group-head"><CheckCircle2 size={15} /><b>已触发</b><span>{triggered.length}</span></div>
          {triggered.length === 0
            ? <p className="alert-empty">尚无触发记录。供应商新报价满足店铺已审核、有货、降幅达标且不高于目标价时会出现在这里。</p>
            : triggered.map((alert) => <AlertRow key={alert.id} alert={alert} />)}
        </div>
      </div>
    </section>
  );
}
