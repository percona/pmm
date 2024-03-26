import axios from 'axios';
import applyCaseMiddleware from 'axios-case-converter';

export const api = applyCaseMiddleware(
  axios.create({
    baseURL: '/v1/',
  })
);
