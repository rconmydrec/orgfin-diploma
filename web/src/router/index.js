import { createRouter, createWebHistory } from 'vue-router';

import HomeView from '@/views/HomeView.vue';
import { useUserStore } from '@/stores/user';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      meta: {
        requiresAuth: false,
        requiresUnAuth: false,
      },
      component: HomeView,
    },
    {
      path: '/login',
      name: 'login',
      meta: {
        requiresAuth: false,
        requiresUnAuth: true,
      },
      component: () => import('@/views/LoginView.vue'),
    },
    {
      path: '/logout',
      name: 'logout',
      meta: {
        requiresAuth: false,
        requiresUnAuth: false,
      },
      component: () => import('@/views/LogoutView.vue'),
    },
    {
      path: '/register',
      name: 'register',
      meta: {
        requiresAuth: false,
        requiresUnAuth: false,
      },
      component: () => import('@/views/RegisterView.vue'),
    },
    {
      path: '/accounts/:id',
      name: 'accountDetails',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('@/views/AccountDetailsView.vue'),
    },
    {
      path: '/activate/:token',
      name: 'userActivate',
      meta: {
        requiresAuth: false,
        requiresUnAuth: false,
      },
      component: () => import('@/views/UserActivateView.vue'),
    },
    {
      path: '/accounts',
      name: 'accounts',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/AccountsListView.vue'),
    },
    {
      path: '/transactions',
      name: 'transactions',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      props: (route) => ({
        categories: route.query.categories,
        startDate: route.query.startDate,
        endDate: route.query.endDate,
      }),
      component: () => import('../views/TransactionsListView.vue'),
    },
    {
      path: '/reports',
      name: 'reports',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/ReportsView.vue'),
    },
    {
      path: '/budgets',
      name: 'budgets',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/BudgetsView.vue'),
    },
    {
      path: '/reports/cashflow',
      name: 'reports-cashflow',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/CashFlowReportView.vue'),
    },
    {
      path: '/reports/balance',
      name: 'reports-balance',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/BalanceReportView.vue'),
    },
    {
      path: '/reports/expenses',
      name: 'reports-expenses',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('@/views/ExpensesReportView.vue'),
    },
    {
      path: '/transactions/:id',
      name: 'transactionDetails',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      props: (route) => ({
        isEdit: true,
        returnUrl: route.query.returnUrl,
        accountId: route.query.accountId,
      }),
      component: () => import('../views/TransactionNewView.vue'),
    },
    {
      path: '/transactions/new',
      name: 'transactionNew',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      props: (route) => ({
        returnUrl: route.query.returnUrl,
        accountId: route.query.accountId,
      }),
      component: () => import('../views/TransactionNewView.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/Settings.vue'),
    },
    {
      path: '/planning',
      name: 'planning',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/FinancialPlanningView.vue'),
    },
    {
      path: '/payment/success',
      name: 'PaymentSuccess',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/PaymentSuccessView.vue'),
    },
    {
      path: '/payment/cancel',
      name: 'PaymentCancel',
      meta: {
        requiresAuth: true,
        requiresUnAuth: false,
      },
      component: () => import('../views/PaymentCancelView.vue'),
    },
  ],
});

router.beforeEach((to) => {
  const userStore = useUserStore();
  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    return { name: 'login' };
  }
  if (to.meta.requiresUnAuth && userStore.isLoggedIn) {
    return { name: 'home' };
  }
});

export default router;
