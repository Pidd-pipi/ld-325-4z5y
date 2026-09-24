export const money = (value: number) => new Intl.NumberFormat('zh-CN', { style: 'currency', currency: 'CNY', maximumFractionDigits: 0 }).format(value);
export const cnDate = (value: string) => new Intl.DateTimeFormat('zh-CN', { month: 'numeric', day: 'numeric' }).format(new Date(value));
export const percent = (value: number) => `${new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 1 }).format(value)}%`;
export const dateTime = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value)) : '';
