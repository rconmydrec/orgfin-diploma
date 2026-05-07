import { request } from './requests';

const budgetsPrefix = '/budgets';

export class BudgetsService {
  userService;

  constructor(userService) {
    this.userService = userService;
  }

  async getUserBudgets(includeType) {
    let budgetUrl = `${budgetsPrefix}/?include=${includeType}`;

    return await request(budgetUrl, {}, { userService: this.userService });
  }

  async createBudget(budget) {
    let budgetUrl = `${budgetsPrefix}/add/`;

    return await request(
      budgetUrl,
      {
        method: 'POST',
        body: JSON.stringify(budget),
      },
      { userService: this.userService },
    );
  }

  async updateBudget(budget) {
    let budgetUrl = `${budgetsPrefix}/${budget.id}/`;

    return await request(
      budgetUrl,
      {
        method: 'PUT',
        body: JSON.stringify(budget),
      },
      { userService: this.userService },
    );
  }

  async deleteBudget(budgetId) {
    let budgetUrl = `${budgetsPrefix}/${budgetId}/`;

    return await request(
      budgetUrl,
      {
        method: 'DELETE',
      },
      { userService: this.userService },
    );
  }

  async archiveBudget(budgetId) {
    let budgetUrl = `${budgetsPrefix}/${budgetId}/archive/`;

    return await request(
      budgetUrl,
      {
        method: 'PUT',
      },
      { userService: this.userService },
    );
  }
}
