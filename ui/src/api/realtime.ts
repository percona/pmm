import {
  RealTimeDataResponse,
  EnableRealTimeAnalyticsRequest,
  DisableRealTimeAnalyticsRequest,
  RealTimeConfig,
} from 'types/realtime.types';
import { api } from './api';

export const getRealTimeData = async (serviceId?: string): Promise<RealTimeDataResponse> => {
  const params = serviceId ? { service_id: serviceId } : {};
  const res = await api.get<RealTimeDataResponse>('/realtime/data', { params });
  return res.data;
};

export const getRealTimeServices = async () => {
  const res = await api.get('/inventory/services');
  const data = res.data;
  
  // Flatten services and filter for MongoDB (only supported type for RTA)
  const services = [];
  if (data.mongodb) {
    services.push(...data.mongodb.map((service: any) => ({
      serviceId: service.service_id,
      serviceName: service.service_name,
      serviceType: 'mongodb',
      nodeId: service.node_id,
      nodeName: service.node_name || '',
      address: service.address,
      port: service.port,
      labels: service.custom_labels || {},
      isEnabled: false, // Default to false, will be determined later
      config: {
        collectionIntervalSeconds: 1,
        disableExamples: false,
      },
      lastSeen: new Date().toISOString(),
    })));
  }
  
  return services;
};

export const enableRealTimeAnalytics = async (
  request: EnableRealTimeAnalyticsRequest
): Promise<void> => {
  await api.post('/realtime/enable', request);
};

export const disableRealTimeAnalytics = async (
  request: DisableRealTimeAnalyticsRequest
): Promise<void> => {
  await api.post('/realtime/disable', request);
};

export const updateRealTimeConfig = async (
  serviceId: string,
  config: RealTimeConfig
): Promise<void> => {
  await api.put(`/realtime/services/${serviceId}/config`, config);
};
