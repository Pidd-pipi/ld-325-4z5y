'use client';

import { BellRing, CheckCircle2, PackageOpen, RefreshCw } from 'lucide-react';

import type { AlertView } from '@/lib/types';
import { dateTime, money, percent } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';

interface AlertCardProps {
  alert: AlertView;
}

const glyph: Record<string, string> = { slab: '▧', floor: '▤', paint: '◒', bath: '◉', hardware: '⌘', window: '▣' };

function PurchaseState({ alert }: { alert: AlertView }) {
  if (!alert.comparable) {
    return <Badge tone="neutral">暂不可比较 · 当前没有有效报价</Badge>;
  }
  const change = alert.current_drop > 0
    ? <em className="drop"> ↓{percent(alert.current_drop)}</em>
    : alert.current_drop < 0
      ? <em> ↑{percent(Math.abs(alert.current_drop))}</em>
      : <em> 价格持平</em>;
  return (
    <span className="purchase-state">
      当前可采购 · {alert.current_supplier} · {money(alert.current_price)}
      {change}
    </span>
  );
}

function ActiveCard({ alert }: AlertCardProps) {
  return (
    <article className="alert-card" key={alert.id}>
      <div className="alert-visual"><span>{glyph[alert.thumbnail] || '▧'}</span></div>
      <div className="alert-body">
        <div className="alert-title">
          <b>{alert.product_name}</b>
          <Badge tone="good">进行中</Badge>
        </div>
        <p>{alert.brand} · {alert.model} · 订阅于 {dateTime(alert.created_at)}</p>
        <div className="alert-figures">
          <span><small>原价（基线）</small><strong>{money(alert.baseline_price)}</strong></span>
          <span><small>目标价</small><strong>{money(alert.target_price)}</strong></span>
          <span><small>降幅要求</small><strong>{percent(alert.drop_percent)}</strong></span>
        </div>
        <PurchaseState alert={alert} />
      </div>
    </article>
  );
}

function TriggeredCard({ alert }: AlertCardProps) {
  return (
    <article className="alert-card triggered" key={alert.id}>
      <div className="alert-visual hit"><span>{glyph[alert.thumbnail] || '▧'}</span></div>
      <div className="alert-body">
        <div className="alert-title">
          <b>{alert.product_name}</b>
          <Badge tone="alert"><CheckCircle2 size={11} /> 已触发</Badge>
        </div>
        <p>{alert.brand} · {alert.model} · 触发时间 {dateTime(alert.triggered_at)}</p>
        <div className="alert-figures">
          <span><small>原价</small><strong>{money(alert.baseline_price)}</strong></span>
          <span><small>触发现价</small><strong className="hit-price">{money(alert.triggered_price)}</strong></span>
          <span><small>降幅</small><strong className="hit-price">↓{percent(((alert.baseline_price - alert.triggered_price) / alert.baseline_price) * 100)}</strong></span>
        </div>
        <p className="trigger-meta">由 {alert.triggered_supplier} 的新报价触发</p>
        <PurchaseState alert={alert} />
      </div>
    </article>
  );
}

interface PersonalCenterProps {
  active: AlertView[];
  triggered: AlertView[];
  loading: boolean;
  onRefresh: () => void;
}

export function PersonalCenter({ active, triggered, loading, onRefresh }: PersonalCenterProps) {
  const empty = !loading && active.length === 0 && triggered.length === 0;
  return (
    <section id="alerts" className="personal-center">
      <div className="section-title">
        <div>
          <p className="eyebrow">PERSONAL CENTER · 价格预警</p>
          <h2>订阅之后，<em>降价会自己找上门。</em></h2>
        </div>
        <div className="catalog-tools">
          <p>基线取订阅时最低的“已审核 + 有货”报价；只有同时满足降幅与目标价的新报价才会触发，同一报价版本只通知一次。</p>
          <button className="refresh-button" onClick={onRefresh} disabled={loading}>
            <RefreshCw size={13} className={loading ? 'spin' : ''} /> 刷新订阅状态
          </button>
        </div>
      </div>

      {empty && (
        <div className="alert-empty">
          <PackageOpen size={22} />
          <b>还没有价格预警订阅</b>
          <span>在下方材料详情中点击「低于 ¥xxx 时提醒」，订阅会出现在这里。</span>
        </div>
      )}

      {active.length > 0 && (
        <div className="alert-group">
          <h3><BellRing size={16} /> 进行中 · {active.length}</h3>
          <div className="alert-grid">
            {active.map((alert) => <ActiveCard key={alert.id} alert={alert} />)}
          </div>
        </div>
      )}

      {triggered.length > 0 && (
        <div className="alert-group">
          <h3><CheckCircle2 size={16} /> 已触发 · {triggered.length}</h3>
          <div className="alert-grid">
            {triggered.map((alert) => <TriggeredCard key={alert.id} alert={alert} />)}
          </div>
        </div>
      )}
    </section>
  );
}
