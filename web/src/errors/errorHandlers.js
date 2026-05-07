import { HttpError } from './HttpError.js';

/**
 * Generic fallback error handler used by views that have no per-form recovery
 * strategy.
 *
 * Behavior:
 *   - 401 → push to `login` (session is gone; logout-style flow).
 *   - Anything else → log only. Do NOT navigate the user away.
 *
 * Historical note: this helper used to unconditionally `router.push({ name: 'home' })`
 * at the end, which destroyed any in-progress form (the user's input was wiped
 * the moment the backend returned 4xx/5xx). That blanket redirect is removed —
 * forms that need to recover from a backend error must catch the error
 * locally, render their own message, and only delegate here for the 401 case.
 *
 * @param {Error} e
 * @param {import('vue-router').Router} router
 */
export async function processError(e, router) {
  if (e instanceof HttpError && e.statusCode === 401) {
    console.log(e.message);
    router.push({ name: 'login' });
    return;
  }
  // Non-401: surface to console for debugging but stay on the current route.
  // Callers that want to display the error to the user must handle it locally.
  console.log(e.message);
}
