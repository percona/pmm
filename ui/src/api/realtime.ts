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
  // Fetch both services and agents to determine RTA status
  const [servicesRes, agentsRes] = await Promise.all([
    api.get('/inventory/services'),
    api.get('/inventory/agents'),
  ]);
  
  const servicesData = servicesRes.data;
  const agentsData = agentsRes.data;
  
  // Create a map of serviceId -> RTA agent for enabled services
  const rtaAgentsByService = new Map<string, any>();
  if (agentsData.mongodb_realtime_analytics_agent) {
    agentsData.mongodb_realtime_analytics_agent.forEach((agent: any) => {
      // Agent is enabled if it's not disabled
      if (!agent.disabled && agent.service_id) {
        rtaAgentsByService.set(agent.service_id, agent);
      }
    });
  }
  
  // Flatten services and filter for MongoDB (only supported type for RTA)
  const services = [];
  if (servicesData.mongodb) {
    services.push(...servicesData.mongodb.map((service: any) => {
      const rtaAgent = rtaAgentsByService.get(service.service_id);
      const isEnabled = !!rtaAgent;
      
      return {
        serviceId: service.service_id,
        serviceName: service.service_name,
        serviceType: 'mongodb',
        nodeId: service.node_id,
        nodeName: service.node_name || '',
        address: service.address,
        port: service.port,
        labels: service.custom_labels || {},
        isEnabled,
        config: rtaAgent ? {
          collectionIntervalSeconds: rtaAgent.realtime_analytics_options?.collection_interval_seconds || 1,
          disableExamples: rtaAgent.realtime_analytics_options?.disable_examples || false,
        } : {
          collectionIntervalSeconds: 1,
          disableExamples: false,
        },
        lastSeen: new Date().toISOString(),
      };
    }));
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
