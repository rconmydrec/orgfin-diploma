import { AccountService } from './accounts';
import { UserService } from './users';
import { CategoriesService } from './categories';
import { TransactionsService } from './transactions';
import { CurrenciesService } from './currencies';
import { SettingsService } from './settings';
import { ReportsService } from '@/services/reports';
import { BudgetsService } from '@/services/budgets';
import { ExportService } from '@/services/export';

import { useUserStore } from '../stores/user';
import { useAccountStore } from '../stores/account';
import { useCategoriesStore } from '../stores/categories';

const userStore = useUserStore();
const accountStore = useAccountStore();
const categoriesStore = useCategoriesStore();

const userService = new UserService(userStore);
const accountsService = new AccountService(accountStore, userService);
const categoriesService = new CategoriesService(categoriesStore, userService);
const transactionsService = new TransactionsService(
  userService,
  accountsService,
);
const currenciesService = new CurrenciesService(userService);
const settingsService = new SettingsService(userService);
const reportsService = new ReportsService(userService);
const budgetsService = new BudgetsService(userService);
const exportService = new ExportService(userService);

userService.injectServices({
  accountsService,
  categoriesService,
  transactionsService,
  currenciesService,
  settingsService,
  reportsService,
  budgetsService,
});

export const Services = {
  userService,
  accountsService,
  categoriesService,
  transactionsService,
  currenciesService,
  settingsService,
  reportsService,
  budgetsService,
  exportService,
};
