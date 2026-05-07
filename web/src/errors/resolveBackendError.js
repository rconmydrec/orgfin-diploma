/**
 * Resolve a parsed backend error body into a user-facing localized string.
 *
 * Resolution priority (per task spec
 * `2026-05-01-1509-transaction-varchar-overflow`):
 *
 *   1. If `body.errorCode` is present AND the i18n key resolves in the current
 *      locale, return the translated string with `body.params` interpolated.
 *   2. Else, if `body.detail` is a non-empty string, return it verbatim
 *      (English fallback from the backend).
 *   3. Else, return the generic `errors.transactionForm.generic` translation.
 *
 * The helper is intentionally backend-agnostic — it inspects `errorCode`,
 * `detail`, and `params`, all of which are part of the project-wide error
 * contract introduced in `backend/internal/handlers/common/responses.go`.
 *
 * @param {{errorCode?: string, detail?: string, params?: object}|null|undefined} body
 *   Parsed JSON response body (typically `HttpError.body`). Tolerates null /
 *   undefined / non-object inputs and falls through to the generic message.
 * @param {(key: string, params?: object) => string} t  vue-i18n `t` function.
 * @param {(key: string) => boolean} [te]
 *   vue-i18n `te` function (key-existence test). When omitted, the helper
 *   conservatively assumes any non-empty `errorCode` resolves — vue-i18n
 *   itself returns the key as a string when missing, which is acceptable but
 *   less precise.
 * @returns {string}
 */
export function resolveBackendError(body, t, te) {
  const code =
    body && typeof body.errorCode === 'string' && body.errorCode.length > 0
      ? body.errorCode
      : null;

  if (code) {
    const params =
      body && body.params && typeof body.params === 'object' ? body.params : {};
    const exists = typeof te === 'function' ? te(code) : true;
    if (exists) {
      const translated = t(code, params);
      // vue-i18n falls back to returning the key itself when it is missing.
      // If we have a `te`, we already filtered that out; otherwise guard here.
      if (translated && translated !== code) {
        return translated;
      }
    }
  }

  if (body && typeof body.detail === 'string' && body.detail.length > 0) {
    return body.detail;
  }

  return t('errors.transactionForm.generic');
}
