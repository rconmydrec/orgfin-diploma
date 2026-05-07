import { toRaw } from 'vue';
import { request } from './requests';
import { useUserStore } from '@/stores/user';

const settingsPrefixUrl = '/settings';

export class SettingsService {
  userService;

  constructor(userService) {
    this.userService = userService;
  }

  async getLanguages() {
    const languagesUrl = `${settingsPrefixUrl}/languages`;
    return await request(
      languagesUrl,
      {},
      { userService: this.userService },
      false,
    );
  }

  async saveUserSettings(settings = null) {
    const saveUserSettingsUrl = `${settingsPrefixUrl}`;
    if (settings === null) {
      settings = toRaw(this.userService.userStore.settings);
    }

    this.updateSettingsInStorage(settings);

    return await request(
      saveUserSettingsUrl,
      {
        method: 'POST',
        body: JSON.stringify(settings),
      },
      { userService: this.userService },
    );
  }

  updateSettingsInStorage(settings) {
    const user = JSON.parse(localStorage.getItem('user'));
    user.settings = settings;
    localStorage.setItem('user', JSON.stringify(user));

    return user;
  }

  async getBaseCurrency() {
    const baseCurrencyUrl = `${settingsPrefixUrl}/base-currency/`;
    return await request(
      baseCurrencyUrl,
      {},
      { userService: this.userService },
    );
  }

  async setBaseCurrency(currency) {
    const userStore = useUserStore();
    const setBaseCurrencyUrl = `${settingsPrefixUrl}/base-currency/`;
    const newCurrency = await request(
      setBaseCurrencyUrl,
      {
        method: 'PUT',
        body: JSON.stringify({
          currencyId: currency.id,
        }),
      },
      { userService: this.userService },
    );

    localStorage.setItem('baseCurrency', newCurrency.code);
    userStore.baseCurrency = newCurrency.code;

    return newCurrency;
  }
}
