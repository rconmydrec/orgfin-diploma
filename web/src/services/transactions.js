import { request } from './requests';

export class TransactionsService {
  userService;
  accountService;

  constructor(userService, accountService) {
    this.userService = userService;
    this.accountService = accountService;
  }

  async getUserTransactions(page = 1, perPage = 20, filters = {}) {
    let transactionsUrl = `/transactions/?per_page=${perPage}&page=${page}`;

    if (filters.accountId && filters.accountId.length > 0) {
      transactionsUrl += `&accounts=${filters.accountId}`;
    }

    if (filters.categories && filters.categories.length > 0) {
      transactionsUrl += `&categories=${filters.categories}`;
    }

    const transactionTypes = Object.keys(filters.transactionTypes).filter(
      (key) => filters.transactionTypes[key] === true,
    );
    if (transactionTypes.length > 0) {
      transactionsUrl += `&types=${transactionTypes.join(',')}`;
    }

    if (filters.fromDate) {
      transactionsUrl += `&from_date=${filters.fromDate}`;
    }

    if (filters.toDate) {
      transactionsUrl += `&to_date=${filters.toDate}`;
    }

    return await request(
      transactionsUrl,
      {},
      { userService: this.userService },
    );
  }

  async getTransactionDetails(id) {
    const transactionDetailsUrl = '/transactions/' + id;
    return await request(
      transactionDetailsUrl,
      {},
      { userService: this.userService },
    );
  }

  async addTransaction(transactionDetails) {
    const transactionDetailsUrl = `/transactions/`;
    const createdTransaction = await request(
      transactionDetailsUrl,
      {
        method: 'POST',
        body: JSON.stringify(transactionDetails),
      },
      { userService: this.userService },
    );
    this.accountService.setShouldUpdateAccountsList(true);
    return createdTransaction;
  }

  async updateTransaction(transactionDetails) {
    const transactionDetailsUrl = `/transactions/`;
    const updatedTransaction = await request(
      transactionDetailsUrl,
      {
        method: 'PUT',
        body: JSON.stringify(transactionDetails),
      },
      { userService: this.userService },
    );
    this.accountService.setShouldUpdateAccountsList(true);
    return updatedTransaction;
  }

  async deleteTransaction(transactionId) {
    const transactionDetailsUrl = `/transactions/${transactionId}`;
    const deletedTransaction = await request(
      transactionDetailsUrl,
      {
        method: 'DELETE',
      },
      { userService: this.userService },
    );
    this.accountService.setShouldUpdateAccountsList(true);
    return deletedTransaction;
  }

  async getUserTemplates() {
    const templatesUrl = '/transactions/templates';
    const templates = await request(
      templatesUrl,
      {},
      { userService: this.userService },
    );
    return templates;
  }

  async updateUserTemplate(templateData) {
    const templatesUrl = '/transactions/templates';
    const updatedTemplate = await request(
      templatesUrl,
      {
        method: 'PUT',
        body: JSON.stringify(templateData),
      },
      { userService: this.userService },
    );

    return updatedTemplate;
  }

  async deleteUserTemplates(templateIds) {
    const templateIdsString = templateIds.join(',');
    const templatesUrl = `/transactions/templates?ids=${templateIdsString}`;
    const deletedTemplates = await request(
      templatesUrl,
      {
        method: 'DELETE',
      },
      { userService: this.userService },
    );

    return deletedTemplates;
  }
}
