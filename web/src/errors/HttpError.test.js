import { describe, it, expect } from 'vitest';
import { HttpError } from './HttpError.js';

describe('HttpError', () => {
  it('preserves the legacy two-argument constructor (message, statusCode)', () => {
    const err = new HttpError('boom', 500);
    expect(err).toBeInstanceOf(Error);
    expect(err.message).toBe('boom');
    expect(err.statusCode).toBe(500);
    expect(err.body).toBeNull();
  });

  it('defaults statusCode to 500 and body to null when omitted', () => {
    const err = new HttpError('boom');
    expect(err.statusCode).toBe(500);
    expect(err.body).toBeNull();
  });

  it('attaches the parsed response body when provided', () => {
    const body = {
      detail: 'Label must be at most 255 characters',
      errorCode: 'errors.transaction.labelTooLong',
      params: { max: 255 },
    };
    const err = new HttpError(body.detail, 422, body);
    expect(err.statusCode).toBe(422);
    expect(err.body).toBe(body);
    expect(err.body.errorCode).toBe('errors.transaction.labelTooLong');
    expect(err.body.params.max).toBe(255);
  });
});
