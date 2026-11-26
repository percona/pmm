import {
  CreateShortUrlRequest,
  CreateShortUrlResponse,
} from 'types/short-urls.types';
import { grafanaApi } from './api';
import { AxiosResponse } from 'axios';

export const createShortUrl = async (path: string) => {
  const response = await grafanaApi.post<
    CreateShortUrlResponse,
    AxiosResponse<CreateShortUrlResponse>,
    CreateShortUrlRequest
  >('/short-urls', { path });
  return response.data;
};
