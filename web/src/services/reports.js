import { request } from './requests';

const reportPrefix = '/reports';
const BACKEND_HOST = import.meta.env.VITE_BACKEND_HOST;

export class ReportsService {
  userService;

  constructor(userService) {
    this.userService = userService;
  }

  async getReport(reportType, filters = {}) {
    let reportUrl = `${reportPrefix}/${reportType}/`;

    return await request(
      reportUrl,
      {
        method: 'POST',
        body: JSON.stringify(filters),
      },
      { userService: this.userService },
    );
  }

  async getDiagram(reportType, startDate, endDate) {
    const reportUrl = `${BACKEND_HOST}${reportPrefix}/diagram/${reportType}/${startDate}/${endDate}`;
    const token = localStorage.getItem('accessToken');

    const response = await fetch(reportUrl, {
      headers: {
        'auth-token': token,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch diagram');
    }

    return await response.json(); // { image: 'data:image/png;base64,...' }
  }

  async getAnalyticsReport(reportType, filters = {}) {
    const analyticsPrefix = '/analytics';
    let reportUrl = `${analyticsPrefix}/${reportType}`;

    return await request(
      reportUrl,
      {
        method: 'POST',
        body: JSON.stringify(filters),
      },
      { userService: this.userService },
    );
  }
}
