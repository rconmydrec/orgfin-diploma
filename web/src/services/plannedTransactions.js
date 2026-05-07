import { request } from './requests';

export class PlannedTransactionsService {
  userService;
  accountService;

  constructor(userService, accountService) {
    this.userService = userService;
    this.accountService = accountService;
  }

  /**
   * Get all planned transactions with optional filters
   * @param {Object} filters - Filter options
   * @param {Array<number>} filters.accountIds - Filter by account IDs
   * @param {string} filters.fromDate - Filter by planned date >= (ISO format)
   * @param {string} filters.toDate - Filter by planned date <= (ISO format)
   * @param {boolean} filters.isRecurring - Filter by recurring flag
   * @param {boolean} filters.isExecuted - Filter by executed flag
   * @param {boolean} filters.isActive - Filter by active flag
   * @param {boolean} filters.includeInactive - Include inactive transactions
   * @returns {Promise<Array>}
   */
  async getPlannedTransactions(filters = {}) {
    let url = '/planned-transactions/';
    const params = new URLSearchParams();

    if (filters.accountIds && filters.accountIds.length > 0) {
      filters.accountIds.forEach((id) => params.append('account_ids', id));
    }

    if (filters.fromDate) {
      params.append('from_date', filters.fromDate);
    }

    if (filters.toDate) {
      params.append('to_date', filters.toDate);
    }

    if (filters.isRecurring !== undefined) {
      params.append('is_recurring', filters.isRecurring);
    }

    if (filters.isExecuted !== undefined) {
      params.append('is_executed', filters.isExecuted);
    }

    if (filters.isActive !== undefined) {
      params.append('is_active', filters.isActive);
    }

    if (filters.includeInactive) {
      params.append('include_inactive', true);
    }

    const queryString = params.toString();
    if (queryString) {
      url += `?${queryString}`;
    }

    return await request(url, {}, { userService: this.userService });
  }

  /**
   * Get a specific planned transaction by ID
   * @param {number} id - Planned transaction ID
   * @returns {Promise<Object>}
   */
  async getPlannedTransaction(id) {
    const url = `/planned-transactions/${id}`;
    return await request(url, {}, { userService: this.userService });
  }

  /**
   * Create a new planned transaction
   * @param {Object} plannedTransactionData - Planned transaction data
   * @param {number} plannedTransactionData.accountId - Account ID
   * @param {number} plannedTransactionData.amount - Amount
   * @param {number|null} plannedTransactionData.categoryId - Category ID (optional)
   * @param {string} plannedTransactionData.label - Label
   * @param {string} plannedTransactionData.notes - Notes
   * @param {boolean} plannedTransactionData.isIncome - Is income flag
   * @param {string} plannedTransactionData.plannedDate - Planned date (ISO format)
   * @param {boolean} plannedTransactionData.isRecurring - Is recurring flag
   * @param {Object|null} plannedTransactionData.recurrenceRule - Recurrence rule (if recurring)
   * @returns {Promise<Object>}
   */
  async createPlannedTransaction(plannedTransactionData) {
    const url = '/planned-transactions/';
    const created = await request(
      url,
      {
        method: 'POST',
        body: JSON.stringify(plannedTransactionData),
      },
      { userService: this.userService },
    );
    return created;
  }

  /**
   * Update a planned transaction
   * @param {number} id - Planned transaction ID
   * @param {Object} plannedTransactionData - Updated planned transaction data
   * @returns {Promise<Object>}
   */
  async updatePlannedTransaction(id, plannedTransactionData) {
    const url = `/planned-transactions/${id}`;
    const updated = await request(
      url,
      {
        method: 'PUT',
        body: JSON.stringify({
          ...plannedTransactionData,
          id,
        }),
      },
      { userService: this.userService },
    );
    return updated;
  }

  /**
   * Delete a planned transaction (soft delete)
   * @param {number} id - Planned transaction ID
   * @returns {Promise<void>}
   */
  async deletePlannedTransaction(id) {
    const url = `/planned-transactions/${id}`;
    await request(
      url,
      {
        method: 'DELETE',
      },
      { userService: this.userService },
    );
  }

  /**
   * Get upcoming transaction occurrences
   * @param {Object} params - Query parameters
   * @param {number} params.days - Number of days to look ahead (default: 30)
   * @param {boolean} params.includeInactive - Include inactive planned transactions
   * @returns {Promise<Array>}
   */
  async getUpcomingOccurrences({ days = 30, includeInactive = false } = {}) {
    let url = '/planned-transactions/upcoming/occurrences';
    const params = new URLSearchParams();

    if (days !== 30) {
      params.append('days', days);
    }

    if (includeInactive) {
      params.append('include_inactive', true);
    }

    const queryString = params.toString();
    if (queryString) {
      url += `?${queryString}`;
    }

    return await request(url, {}, { userService: this.userService });
  }

  /**
   * Calculate future balance on a target date
   * @param {Object} params - Calculation parameters
   * @param {string} params.targetDate - Target date (ISO format)
   * @param {Array<number>|null} params.accountIds - Account IDs (optional, null = all accounts)
   * @param {boolean} params.includeInactive - Include inactive planned transactions
   * @returns {Promise<Object>}
   */
  async calculateFutureBalance({
    targetDate,
    accountIds = null,
    includeInactive = false,
    includeHidden = false,
  }) {
    const url = '/financial-planning/future-balance';
    return await request(
      url,
      {
        method: 'POST',
        body: JSON.stringify({
          targetDate,
          accountIds,
          includeInactive,
          includeHidden,
        }),
      },
      { userService: this.userService },
    );
  }

  /**
   * Get balance projection over a time period
   * @param {Object} params - Projection parameters
   * @param {string} params.startDate - Start date (ISO format, defaults to now)
   * @param {string} params.endDate - End date (ISO format)
   * @param {string} params.period - Period: 'daily', 'weekly', or 'monthly'
   * @param {Array<number>|null} params.accountIds - Account IDs (optional, null = all accounts)
   * @param {boolean} params.includeInactive - Include inactive planned transactions
   * @returns {Promise<Object>}
   */
  async getBalanceProjection({
    startDate = new Date().toISOString(),
    endDate,
    period = 'daily',
    accountIds = null,
    includeInactive = false,
    includeHidden = false,
  }) {
    const url = '/financial-planning/projection';
    return await request(
      url,
      {
        method: 'POST',
        body: JSON.stringify({
          startDate,
          endDate,
          period,
          accountIds,
          includeInactive,
          includeHidden,
        }),
      },
      { userService: this.userService },
    );
  }
}
