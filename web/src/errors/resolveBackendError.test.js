import { describe, it, expect, vi } from 'vitest';
import { resolveBackendError } from './resolveBackendError.js';

/** Fake i18n `t` that interpolates `{name}` placeholders from `params`. */
function makeT(translations) {
  return (key, params = {}) => {
    if (!Object.prototype.hasOwnProperty.call(translations, key)) {
      // Mirror vue-i18n behaviour: missing key is returned verbatim.
      return key;
    }
    let out = translations[key];
    for (const [k, v] of Object.entries(params)) {
      out = out.replaceAll(`{${k}}`, String(v));
    }
    return out;
  };
}

function makeTe(translations) {
  return (key) => Object.prototype.hasOwnProperty.call(translations, key);
}

const TRANSLATIONS = {
  'errors.transaction.labelTooLong': 'Label must be at most {max} characters',
  'errors.transaction.notesTooLong': 'Notes must be at most {max} characters',
  'errors.transaction.validationFailed': 'Validation failed',
  'errors.transactionForm.generic':
    'An unexpected error occurred — please try again.',
};

describe('resolveBackendError', () => {
  it('uses the i18n key from errorCode and interpolates params', () => {
    const t = makeT(TRANSLATIONS);
    const te = makeTe(TRANSLATIONS);
    const result = resolveBackendError(
      {
        detail: 'Label must be at most 255 characters',
        errorCode: 'errors.transaction.labelTooLong',
        params: { max: 255 },
      },
      t,
      te,
    );
    expect(result).toBe('Label must be at most 255 characters');
  });

  it('falls back to detail when errorCode is missing', () => {
    const t = makeT(TRANSLATIONS);
    const te = makeTe(TRANSLATIONS);
    const result = resolveBackendError(
      { detail: 'Some custom server message' },
      t,
      te,
    );
    expect(result).toBe('Some custom server message');
  });

  it('falls back to detail when errorCode is unknown to the locale', () => {
    const t = makeT(TRANSLATIONS);
    const te = makeTe(TRANSLATIONS);
    const result = resolveBackendError(
      {
        detail: 'Server-side fallback',
        errorCode: 'errors.unknown.something',
      },
      t,
      te,
    );
    expect(result).toBe('Server-side fallback');
  });

  it('falls back to the generic key when neither errorCode nor detail is usable', () => {
    const t = makeT(TRANSLATIONS);
    const te = makeTe(TRANSLATIONS);
    expect(resolveBackendError(null, t, te)).toBe(
      'An unexpected error occurred — please try again.',
    );
    expect(resolveBackendError({}, t, te)).toBe(
      'An unexpected error occurred — please try again.',
    );
    expect(resolveBackendError({ detail: '' }, t, te)).toBe(
      'An unexpected error occurred — please try again.',
    );
  });

  it('tolerates a missing `te` argument', () => {
    const t = makeT(TRANSLATIONS);
    const spy = vi.fn(t);
    const result = resolveBackendError(
      {
        errorCode: 'errors.transaction.notesTooLong',
        params: { max: 4000 },
        detail: 'Notes must be at most 4000 characters',
      },
      spy,
    );
    expect(result).toBe('Notes must be at most 4000 characters');
    expect(spy).toHaveBeenCalledWith('errors.transaction.notesTooLong', {
      max: 4000,
    });
  });
});
