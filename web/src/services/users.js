import { HttpError } from '../errors/HttpError';
import { request } from './requests';
import router from '../router';

export class UserService {
  userStore;
  accountsService;
  categoriesService;

  constructor(userStore) {
    this.userStore = userStore;
  }

  injectServices(services = {}) {
    for (const [serviceName, service] of Object.entries(services)) {
      this[serviceName] = service;
    }
  }

  async loginUser(loginEmail, password) {
    const loginPath = '/auth/login/';
    const requestBody = {
      email: loginEmail,
      password,
    };

    await this.tryToLogin(loginPath, requestBody);
  }

  async activateUser(token) {
    const activatePath = '/auth/activate/';

    return await request(
      `${activatePath}${token}`,
      { userService: this },
      false,
      false,
    );
  }

  async getUserProfile(accessToken) {
    try {
      const profileEndpoint = '/auth/profile/';
      const userProfile = await request(
        profileEndpoint,
        {},
        { userService: this },
      );
      this.setUser(userProfile, true, accessToken);
    } catch (e) {
      console.log(e);
    }
  }

  setUser(userProfile, isLoggedIn = false, accessToken = '') {
    // need to make deep copying to avoid cycling errors
    const clonedUserProfile = JSON.parse(JSON.stringify(userProfile));
    this.userStore.user = Object.assign(this.userStore.user, clonedUserProfile);

    this.userStore.isLoggedIn = isLoggedIn;
    this.userStore.accessToken = accessToken;
    this.userStore.settings = Object.assign(
      this.userStore.settings,
      userProfile.settings,
    );
    this.userStore.baseCurrency = userProfile.baseCurrency;

    localStorage.setItem('user', JSON.stringify(this.userStore.user));
    localStorage.setItem('isLoggedIn', this.userStore.isLoggedIn);
    localStorage.setItem('accessToken', this.userStore.accessToken);
    localStorage.setItem('settings', JSON.stringify(this.userStore.settings));
    localStorage.setItem('baseCurrency', this.userStore.baseCurrency);
  }

  async logOutUser() {
    this.userStore.stopTimer();
    this.userStore.isLoggedIn = false;
    this.userStore.accessToken = '';
    this.setUser(
      {
        id: null,
        first_name: null,
        last_name: null,
        email: null,
        iat: null,
        exp: null,
      },
      false,
      '',
    );
    this.accountsService.clearAccounts();
    localStorage.clear();
    await router.push({ name: 'login' });
  }

  async registerUser(email, password, firstName, lastName) {
    const registerPath = '/auth/register/';
    const requestBody = {
      email,
      password,
      first_name: firstName,
      last_name: lastName,
    };
    try {
      await request(
        registerPath,
        {
          method: 'POST',
          body: JSON.stringify(requestBody),
        },
        { userService: this },
        false,
      );
    } catch (e) {
      if (e instanceof HttpError) {
        console.error('Registration error:', e.message);
      } else {
        console.error('Unexpected error:', e);
      }
      throw e;
    }
  }

  async oauthLogin(credential) {
    const loginPath = '/auth/oauth/';
    const requestBody = {
      credential: credential,
    };

    await this.tryToLogin(loginPath, requestBody);
  }

  async tryToLogin(loginPath, requestBody) {
    try {
      const data = await request(
        loginPath,
        {
          method: 'POST',
          body: JSON.stringify(requestBody),
        },
        { userService: this },
        false,
      );

      if (data.accessToken) {
        this.userStore.accessToken = data.accessToken;
        this.userStore.startTimer();
        await this.getUserProfile(this.userStore.accessToken);
        await this.accountsService.getUserAccounts({});
        await this.categoriesService.getUserCategories(true);
      } else {
        console.log('No access token in response!');
      }
    } catch (e) {
      if (e instanceof HttpError && e.statusCode === 401) {
        if (e.message === 'User not activated') {
          console.log('User not activated!');
        } else {
          console.log('Wrong email or password!');
        }
      }
      throw e;
    }
  }

  async changePassword(currentPassword, newPassword) {
    const changePasswordPath = '/auth/change-password/';
    const requestBody = {
      currentPassword,
      newPassword,
    };

    try {
      const data = await request(
        changePasswordPath,
        {
          method: 'POST',
          body: JSON.stringify(requestBody),
        },
        { userService: this },
        true,
      );
      return data;
    } catch (e) {
      if (e instanceof HttpError) {
        console.error('Change password error:', e.message);
      } else {
        console.error('Unexpected error:', e);
      }
      throw e;
    }
  }
}
