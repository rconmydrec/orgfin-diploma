import { request } from './requests';
import { DateTime } from 'luxon';

const categoriesUrlPrefix = '/categories';

export class CategoriesService {
  userService;
  categoriesStore;

  timeToCacheCategories = 600000 * 6; // 60 minutes

  constructor(categoriesStore, userService) {
    this.userService = userService;
    this.categoriesStore = categoriesStore;
  }

  setShouldUpdateCategories(shouldUpdate) {
    this.categoriesStore.shouldUpdate = shouldUpdate;
  }

  async getUserCategories(shouldUpdate = false) {
    const currentTime = DateTime.now();
    const deltaFromLastUpdate = currentTime - this.categoriesStore.lastUpdated;

    if (shouldUpdate) {
      this.setShouldUpdateCategories(true);
    }

    if (
      !this.categoriesStore.shouldUpdate &&
      this.categoriesStore.categories.length > 0 &&
      deltaFromLastUpdate < this.timeToCacheCategories
    ) {
      return this.categoriesStore.categories;
    }

    const categoriesUrl = `${categoriesUrlPrefix}/`;
    const categories = await request(
      categoriesUrl,
      {},
      { userService: this.userService },
    );
    if (categories) {
      this.updateCategories(categories);
      return this.categoriesStore.categories;
    } else {
      return [];
    }
  }

  async getGroupedCategories() {
    const categoriesUrl = `${categoriesUrlPrefix}/grouped/`;
    return await request(categoriesUrl, {}, { userService: this.userService });
  }

  updateCategories(categories) {
    this.categoriesStore.categories.length = 0;
    this.categoriesStore.categories.push(...categories);
    this.categoriesStore.lastUpdated = DateTime.now();
    this.setShouldUpdateCategories(false);
  }

  async updateCategory(category) {
    const categoriesUrl = `${categoriesUrlPrefix}/${category.id}/`;
    const updatedCategory = await request(
      categoriesUrl,
      {
        method: 'PUT',
        body: JSON.stringify(category),
      },
      { userService: this.userService },
    );

    await this.getUserCategories(true);

    return updatedCategory;
  }

  async createCategory(category) {
    const categoriesUrl = `${categoriesUrlPrefix}/`;
    const newCategory = await request(
      categoriesUrl,
      {
        method: 'POST',
        body: JSON.stringify(category),
      },
      { userService: this.userService },
    );

    await this.getUserCategories(true);

    return newCategory;
  }

  async deleteCategory(categoryId) {
    const categoriesUrl = `${categoriesUrlPrefix}/${categoryId}/`;
    const deletedCategory = await request(
      categoriesUrl,
      {
        method: 'DELETE',
      },
      { userService: this.userService },
    );

    await this.getUserCategories(true);

    return deletedCategory;
  }
}
