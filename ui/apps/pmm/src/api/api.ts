import axios from 'axios';
import applyCaseMiddleware from 'axios-case-converter';

export const api = applyCaseMiddleware(
  axios.create({
    baseURL: '/v1/',
  })
);

export const grafanaApi = axios.create({
  baseURL: '/graph/api/',
});
