import { request } from './requests';

const BACKEND_HOST = import.meta.env.VITE_BACKEND_HOST;

export class ExportService {
  userService;

  constructor(userService) {
    this.userService = userService;
  }

  async downloadExport(startDate, endDate) {
    const token = localStorage.getItem('accessToken');
    const response = await fetch(`${BACKEND_HOST}/export/download`, {
      method: 'POST',
      headers: {
        'auth-token': token,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ start_date: startDate, end_date: endDate }),
    });

    if (!response.ok) {
      const err = await response.json().catch(() => ({}));
      throw new Error(err.detail || 'Failed to download export');
    }

    const blob = await response.blob();
    const disposition = response.headers.get('Content-Disposition');
    let filename = 'transactions.xlsx';
    if (disposition) {
      const match = disposition.match(/filename="?([^";\n]+)"?/);
      if (match) {
        filename = match[1];
      }
    }

    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
  }

  async emailExport(startDate, endDate) {
    return await request(
      '/export/email',
      {
        method: 'POST',
        body: JSON.stringify({ start_date: startDate, end_date: endDate }),
      },
      { userService: this.userService },
    );
  }
}
