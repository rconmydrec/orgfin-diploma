import { describe, it, expect, vi } from 'vitest';
import { processError } from './errorHandlers.js';
import { HttpError } from './HttpError.js';

function makeRouter() {
  return { push: vi.fn() };
}

describe('processError', () => {
  it('redirects to login on a 401 HttpError', async () => {
    const router = makeRouter();
    await processError(new HttpError('Unauthorized', 401), router);
    expect(router.push).toHaveBeenCalledWith({ name: 'login' });
    expect(router.push).toHaveBeenCalledTimes(1);
  });

  it('does NOT redirect to home on a 422 validation error', async () => {
    const router = makeRouter();
    await processError(
      new HttpError('Label must be at most 255 characters', 422, {
        errorCode: 'errors.transaction.labelTooLong',
      }),
      router,
    );
    expect(router.push).not.toHaveBeenCalled();
  });

  it('does NOT redirect to home on a 500 error', async () => {
    const router = makeRouter();
    await processError(new HttpError('Server error', 500), router);
    expect(router.push).not.toHaveBeenCalled();
  });

  it('does NOT redirect on a non-HttpError', async () => {
    const router = makeRouter();
    await processError(new Error('boom'), router);
    expect(router.push).not.toHaveBeenCalled();
  });
});
