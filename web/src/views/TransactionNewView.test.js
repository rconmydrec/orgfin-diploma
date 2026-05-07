import { describe, it, expect, beforeEach, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { createRouter, createMemoryHistory } from 'vue-router';

import { HttpError } from '@/errors/HttpError.js';
import enMessages from '@/locales/en.json';

// --- Module mocks ----------------------------------------------------------

const transactionsServiceMock = {
  addTransaction: vi.fn(),
  updateTransaction: vi.fn(),
  getUserTemplates: vi.fn().mockResolvedValue([]),
  getTransactionDetails: vi.fn(),
  deleteTransaction: vi.fn(),
};

const accountsServiceMock = {
  getUserAccounts: vi.fn().mockResolvedValue([
    {
      id: 1,
      name: 'Cash',
      currency: { code: 'USD' },
    },
    {
      id: 2,
      name: 'Bank',
      currency: { code: 'USD' },
    },
  ]),
};

const categoriesServiceMock = {
  getUserCategories: vi
    .fn()
    .mockResolvedValue([{ id: 10, name: 'Food', isIncome: false }]),
};

vi.mock('@/services/servicesConfig', () => ({
  Services: {
    transactionsService: transactionsServiceMock,
    accountsService: accountsServiceMock,
    categoriesService: categoriesServiceMock,
  },
}));

vi.mock('@/stores/user', () => ({
  useUserStore: () => ({
    transactionTemplates: [],
    accessToken: 'test-token',
  }),
}));

// Spy on the global `processError` so we can assert when it is and is not
// called. Re-exports stay live so the actual implementation runs (we want to
// observe its router.push behavior end-to-end).
import * as errorHandlers from '@/errors/errorHandlers.js';

// Stub the visually-heavy child components — they are not under test.
const stubChild = {
  template: '<div></div>',
  props: {
    transaction: { type: Object, default: () => ({}) },
    isEdit: { type: Boolean, default: false },
    isTransfer: { type: Boolean, default: false },
    accounts: { type: Array, default: () => [] },
    accountId: { type: [Number, String, null], default: null },
    accountType: { type: String, default: '' },
    label: { type: String, default: '' },
    amount: { type: [Number, String, null], default: null },
    currentAccount: { type: Object, default: () => ({}) },
    type: { type: String, default: '' },
    amountInput: { type: Object, default: null },
    itemType: { type: String, default: '' },
    categories: { type: Array, default: () => [] },
    currencySrcCode: { type: String, default: '' },
    srcAmount: { type: [Number, String, null], default: null },
    currencyTargetCode: { type: String, default: '' },
    targetAmount: { type: [Number, String, null], default: null },
  },
};

// --- Helpers ---------------------------------------------------------------

async function mountView() {
  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    fallbackLocale: 'en',
    messages: { en: enMessages },
  });

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'home', component: { template: '<div />' } },
      {
        path: '/transactions',
        name: 'transactions',
        component: { template: '<div />' },
      },
      {
        path: '/account/:id',
        name: 'accountDetails',
        component: { template: '<div />' },
      },
      { path: '/login', name: 'login', component: { template: '<div />' } },
      { path: '/new', name: 'new', component: { template: '<div />' } },
    ],
  });
  router.push('/new');
  await router.isReady();

  // Lazy-import the component AFTER mocks are registered so the mocked
  // modules are picked up.
  const { default: TransactionNewView } = await import(
    './TransactionNewView.vue'
  );

  const wrapper = mount(TransactionNewView, {
    props: { isEdit: false, returnUrl: 'transactions', accountId: null },
    global: {
      plugins: [i18n, router],
      stubs: {
        TransactionTypeTabs: stubChild,
        TransactionLabel: stubChild,
        TransactionAmount: stubChild,
        Category: stubChild,
        AccountSelector: stubChild,
        ExchangeRate: stubChild,
        TransactionDateTime: stubChild,
      },
    },
  });

  // Wait for onBeforeMount async setup (categories, accounts, templates).
  await flushPromises();
  return { wrapper, router };
}

function findBanner(wrapper) {
  return wrapper.find('[data-test="transaction-form-error"]');
}

// --- Tests -----------------------------------------------------------------

describe('TransactionNewView — backend error UX', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('on 422 with errorCode: keeps the form mounted, shows the localized banner, does not navigate', async () => {
    const body = {
      detail: 'Label must be at most 255 characters',
      errorCode: 'errors.transaction.labelTooLong',
      params: { max: 255 },
    };
    transactionsServiceMock.addTransaction.mockRejectedValueOnce(
      new HttpError(body.detail, 422, body),
    );

    const { wrapper, router } = await mountView();
    const routerPush = vi.spyOn(router, 'push');
    const processErrorSpy = vi.spyOn(errorHandlers, 'processError');

    await wrapper.find('form').trigger('submit.prevent');
    await flushPromises();

    expect(wrapper.find('form').exists()).toBe(true);
    const banner = findBanner(wrapper);
    expect(banner.exists()).toBe(true);
    expect(banner.text()).toBe('Label must be at most 255 characters');
    expect(banner.classes()).toContain('alert');
    expect(banner.classes()).toContain('alert-danger');

    // Overlay dismissed.
    expect(wrapper.find('#submitted-overlay').exists()).toBe(false);

    // No route change, no delegation to the global handler.
    expect(routerPush).not.toHaveBeenCalled();
    expect(processErrorSpy).not.toHaveBeenCalled();
  });

  it('on 500: keeps the form mounted, shows the generic banner, does not navigate', async () => {
    transactionsServiceMock.addTransaction.mockRejectedValueOnce(
      new HttpError('An unexpected error occurred', 500, {
        detail: 'An unexpected error occurred',
      }),
    );

    const { wrapper, router } = await mountView();
    const routerPush = vi.spyOn(router, 'push');

    await wrapper.find('form').trigger('submit.prevent');
    await flushPromises();

    expect(wrapper.find('form').exists()).toBe(true);
    const banner = findBanner(wrapper);
    expect(banner.exists()).toBe(true);
    // Falls back to `detail` since no errorCode key resolves.
    expect(banner.text()).toBe('An unexpected error occurred');

    expect(wrapper.find('#submitted-overlay').exists()).toBe(false);
    expect(routerPush).not.toHaveBeenCalled();
  });

  it('on 401: delegates to processError and pushes to login', async () => {
    transactionsServiceMock.addTransaction.mockRejectedValueOnce(
      new HttpError('Unauthorized', 401),
    );

    const { wrapper, router } = await mountView();
    const routerPush = vi.spyOn(router, 'push');

    await wrapper.find('form').trigger('submit.prevent');
    await flushPromises();

    expect(routerPush).toHaveBeenCalledWith({ name: 'login' });
    // Banner is NOT rendered for 401 — the user is redirected.
    expect(findBanner(wrapper).exists()).toBe(false);
  });

  it('clears the banner when the user edits the notes field', async () => {
    transactionsServiceMock.addTransaction.mockRejectedValueOnce(
      new HttpError('Validation failed', 422, {
        detail: 'Validation failed',
        errorCode: 'errors.transaction.validationFailed',
      }),
    );

    const { wrapper } = await mountView();
    await wrapper.find('form').trigger('submit.prevent');
    await flushPromises();
    expect(findBanner(wrapper).exists()).toBe(true);

    await wrapper.find('textarea#notes').setValue('a note');
    await flushPromises();
    expect(findBanner(wrapper).exists()).toBe(false);
  });
});
