export class HttpError extends Error {
  /** @type {number} */
  statusCode;

  /**
   * Parsed JSON response body (when the response body was JSON-decodable).
   * Carries structured error fields like `detail`, `errorCode`, `params` so
   * callers can render localized messages without re-parsing the response.
   * Null when the body could not be parsed or no body was attached.
   *
   * @type {object|null}
   */
  body;

  constructor(message, statusCode = 500, body = null) {
    super(message);
    this.statusCode = statusCode;
    this.body = body;
  }
}
