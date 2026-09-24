'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { ArrowDownRight, LoaderCircle, ShieldCheck } from 'lucide-react';
import { api } from '@/lib/api';
import type { AlertGroups, Product, Supplier, Trend } from '@/lib/types';
import { AppHeader } from '@/components/AppHeader';
import { CategoryRail } from '@/components/CategoryRail';
import { ProductCatalog } from '@/components/ProductCatalog';
import { ComparisonTray } from '@/components/ComparisonTray';
import { InsightPanel } from '@/components/InsightPanel';
import { BudgetCalculator } from '@/components/BudgetCalculator';
import { PersonalCenter } from '@/components/PersonalCenter';
import { SupplierConsole } from '@/components/SupplierConsole';

const emptyAlerts: AlertGroups = { active: [], triggered: [] };

export default function Home() {
  const [products, setProducts] = useState<Product[]>([]);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [alerts, setAlerts] = useState<AlertGroups>(emptyAlerts);
  const [alertsLoading, setAlertsLoading] = useState(false);
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('全部');
  const [compareIDs, setCompareIDs] = useState<number[]>([]);
  const [selected, setSelected] = useState<Product | null>(null);
  const [trend, setTrend] = useState<Trend | null>(null);
  const [trendRange, setTrendRange] = useState<'30d' | '90d' | '1y'>('30d');
  const [sort, setSort] = useState<'price' | 'sales' | 'rating'>('rating');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(true);
  const [offerBusy, setOfferBusy] = useState(false);

  const notify = useCallback((text: string) => {
    setMessage(text);
    window.setTimeout(() => setMessage(''), 3200);
  }, []);

  const reloadAlerts = useCallback(async (silent = false) => {
    if (!silent) setAlertsLoading(true);
    try {
      setAlerts(await api.alerts());
    } catch {
      if (!silent) setAlerts(emptyAlerts);
    } finally {
      setAlertsLoading(false);
    }
  }, []);

  useEffect(() => {
    Promise.all([api.listProducts(), api.listSuppliers(), api.alerts()])
      .then(([catalog, shops, alertData]) => {
        setProducts(catalog.items);
        setSuppliers(shops);
        setSelected(catalog.items[0] || null);
        setAlerts(alertData);
      })
      .catch(() => setMessage('报价数据暂时不可用，请确认后端服务已启动。'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (selected) api.trend(selected.ID, trendRange).then(setTrend).catch(() => setTrend(null));
  }, [selected, trendRange]);

  const visible = useMemo(() => products.filter((item) => (category === '全部' || item.Category?.Name === category) && `${item.Name}${item.Brand}${item.Model}`.toLowerCase().includes(query.toLowerCase())).sort((a, b) => sort === 'price' ? Math.min(...a.Offers.map((offer) => offer.UnitPrice)) - Math.min(...b.Offers.map((offer) => offer.UnitPrice)) : sort === 'sales' ? b.SalesCount - a.SalesCount : b.Rating - a.Rating), [products, category, query, sort]);
  const compare = (id: number) => setCompareIDs((current) => current.includes(id) ? current.filter((value) => value !== id) : current.length < 4 ? [...current, id] : current);

  return (
    <main id="top">
      <AppHeader query={query} onQuery={setQuery} alertCount={alerts.triggered.length} />
      <section className="hero">
        <div>
          <p className="eyebrow">MATERIAL MARKET INTELLIGENCE / SINCE 2026</p>
          <h1>不是找最低价。<br /><em>是买到恰好的那一笔。</em></h1>
          <p className="hero-copy">把品牌、规格、交期与多商家报价放到同一张桌子上。今天的采购，应该有据可依。</p>
          <div className="hero-note"><ShieldCheck size={18} /><span>订阅基线 · 降价触发 · 同版本不重复通知 · 真实可比较字段</span></div>
        </div>
        <div className="hero-number"><span>本期已收录</span><strong>8,624</strong><b>条有效报价 <ArrowDownRight size={18} /></b><p>覆盖瓷砖、地板、涂料、卫浴等<br />装修决策中的高频材料。</p></div>
      </section>
      <CategoryRail selected={category} onSelected={setCategory} />
      {loading ? <div className="loading"><LoaderCircle className="spin" /> 正在汇总市场报价…</div> : (
        <>
          <ProductCatalog
            products={visible}
            compareIDs={compareIDs}
            onCompare={compare}
            onFavorite={async (id) => { await api.favorite(id); notify('已收入「本周采购」收藏夹'); }}
            onSelect={setSelected}
            sort={sort}
            onSort={setSort}
          />
          <ComparisonTray items={products.filter((item) => compareIDs.includes(item.ID))} onRemove={compare} />
          <InsightPanel
            product={selected}
            trend={trend}
            range={trendRange}
            onRange={setTrendRange}
            onAlert={async (id, target, dropPercent) => {
              await api.alert(id, target, dropPercent);
              notify('价格预警已建立，基线已保存；满足降幅与目标价时会在这里通知');
              await reloadAlerts(true);
            }}
          />
          <SupplierConsole
            products={products}
            suppliers={suppliers}
            busy={offerBusy}
            onSubmit={async (input) => {
              setOfferBusy(true);
              try {
                const result = await api.submitOffer(input);
                const [catalog] = await Promise.all([api.listProducts(), reloadAlerts(true)]);
                setProducts(catalog.items);
                setSelected((current) => catalog.items.find((item) => item.ID === current?.ID) || current);
                if (result.triggered_count > 0) {
                  notify(`新报价已提交，触发 ${result.triggered_count} 条价格预警`);
                } else if (result.new_version) {
                  notify('新报价已提交，暂未达到订阅的降幅或目标价');
                } else {
                  notify('相同报价版本重复提交，未生成新记录，也不会重复通知');
                }
              } catch (error) {
                notify(error instanceof Error ? error.message : '报价提交失败');
              } finally {
                setOfferBusy(false);
              }
            }}
          />
          <PersonalCenter
            active={alerts.active}
            triggered={alerts.triggered}
            loading={alertsLoading}
            onRefresh={() => reloadAlerts()}
          />
          <BudgetCalculator onSubmit={async (room, area) => { const result = await api.budget(room, area); notify('预算已保存，可继续替换为实际报价'); return result.Estimate; }} />
        </>
      )}
      {message && <div className="toast" role="status">{message}</div>}
      <footer><span>筑价 BUILD PRICE INDEX</span><span>报价仅作采购决策参考，请以商家最终合同为准。</span></footer>
    </main>
  );
}
