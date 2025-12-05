export interface CreateShortUrlRequest {
  path: string;
}

export interface CreateShortUrlResponse {
  uid: string;
  url: string;
}
