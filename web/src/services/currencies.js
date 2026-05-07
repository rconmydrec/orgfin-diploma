import { request } from './requests';

export class CurrenciesService {
  _userService;

  constructor(userService) {
    this._userService = userService;
  }

  async getAllCurrencies() {
    const currenciesUrl = '/currencies/';
    return await request(currenciesUrl, {}, { userService: this._userService });
  }

  async getCurrencyById(id) {
    const currencyUrl = `/currencies/${id}`;
    return await request(currencyUrl, {}, { userService: this._userService });
  }
}
