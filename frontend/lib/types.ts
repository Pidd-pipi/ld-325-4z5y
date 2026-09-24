export type Supplier = { ID: number; Name: string; Rating: number; Address: string; Contact?: string; Status: string };
export type Offer = { ID: number; UnitPrice: number; MOQ: number; Freight: string; DeliveryDays: number; StockStatus: string; Supplier: Supplier };
export type Product = { ID: number; Name: string; Brand: string; Model: string; Unit: string; Thumbnail: string; SalesCount: number; Rating: number; Category: { Name: string }; Offers: Offer[] };
export type ApiEnvelope<T> = { code: number; message: string; data: T };
export type TrendPoint = { Price: number; RecordedAt: string };
export type Trend = { range: string; highest: number; lowest: number; average: number; points: TrendPoint[] };

export type AlertView = {
  id: number;
  status: 'active' | 'triggered';
  product_id: number;
  product_name: string;
  brand: string;
  model: string;
  unit: string;
  thumbnail: string;
  target_price: number;
  drop_percent: number;
  baseline_price: number;
  current_price: number;
  current_supplier: string;
  current_drop: number;
  comparable: boolean;
  triggered_price: number;
  triggered_supplier: string;
  triggered_at: string;
  created_at: string;
};

export type AlertGroups = {
  active: AlertView[];
  triggered: AlertView[];
};

export type OfferSubmission = {
  product_id: number;
  supplier_id: number;
  unit_price: number;
  moq: number;
  freight: string;
  delivery_days: number;
  stock_status: 'in_stock' | 'out_of_stock' | 'discontinued';
};

export type OfferSubmissionResult = {
  new_version: boolean;
  triggered_count: number;
};
