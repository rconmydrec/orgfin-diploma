/*
just a wrapper to add auth header to each request
*/

import { HttpError } from '../errors/HttpError';
import { useUserStore } from '../stores/user';

const BACKEND_HOST = import.meta.env.VITE_BACKEND_HOST;

/**
 * Read the response body as JSON if possible. Returns `null` when the body
 * is empty or not valid JSON — callers must tolerate either case.
 *
 * @param {Response} response
 * @returns {Promise<object|null>}
 */
async function safeJson(response) {
  try {
    const text = await response.text();
    if (!text) {
      return null;
    }
    return JSON.parse(text);
  } catch {
    return null;
  }
}

export async function request(
  endPoint,
  params = {},
  services = {},
  authRequired = true,
) {
  const userStore = useUserStore();

  let accessToken = userStore.accessToken;
  if (authRequired) {
    if (accessToken === null) {
      accessToken = localStorage.getItem('accessToken');
    }
    if (accessToken === null) {
      throw new HttpError('Unauthorized', 401);
    }
  }
  const defaultHeaders = {
    'auth-token': accessToken,
    'Content-Type': 'application/json',
  };
  params.headers = { ...defaultHeaders, ...params.headers };

  const response = await fetch(`${BACKEND_HOST}${endPoint}`, params);

  if ([200, 201, 202, 204].includes(response.status)) {
    const newAccessToken = response.headers.get('new_access_token');
    if (newAccessToken) {
      userStore.accessToken = newAccessToken;
      localStorage.setItem('accessToken', newAccessToken);
    }
    try {
      if (response.status === 204) {
        return null;
      } else {
        return await response.json();
      }
    } catch (e) {
      throw new HttpError('An unexpected error occurred', response.status);
    }
  } else {
    const body = await safeJson(response);
    const detail =
      (body && typeof body.detail === 'string' && body.detail) || null;

    if (response.status === 401) {
      await services.userService.logOutUser();
      if (detail === 'User not activated') {
        throw new HttpError('User not activated', response.status, body);
      }
      throw new HttpError('Unauthorized', response.status, body);
    }

    throw new HttpError(
      detail || 'An unexpected error occurred',
      response.status,
      body,
    );
  }
}
