export const CREDIT_CARD_ACC_TYPE_ID = 4;
export const INVESTMENT_ACC_TYPE_ID = 6;

export const ACCOUNT_TYPE_CONFIG = {
  1: { icon: 'fa-solid fa-money-bill-wave', color: 'success', key: 'cash' },
  2: {
    icon: 'fa-solid fa-building-columns',
    color: 'primary',
    key: 'regular_bank',
  },
  3: { icon: 'fa-regular fa-credit-card', color: 'info', key: 'debit_card' },
  4: { icon: 'fa-solid fa-credit-card', color: 'warning', key: 'credit_card' },
  5: { icon: 'fa-solid fa-file-invoice-dollar', color: 'danger', key: 'loan' },
  6: { icon: 'fa-solid fa-chart-line', color: 'purple', key: 'investment' },
};
