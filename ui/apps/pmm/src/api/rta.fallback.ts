/**
 * @fileoverview
 * This file contains the fallback API for the RTA API.
 * It is used to get the running real-time agents and change the real-time agent status.
 * It is used to fallback to the old API until the new API is merged.
 */

import { AgentStatus } from "types/agent.types";
import { api } from "./api";

interface ListRunningRealTimeAgentsResponse {
    agents: RunningRealTimeAgent[];
}

interface RunningRealTimeAgent {
    agentId: string;
    serviceId: string;
    serviceName: string;
    cluster: string;
    startedAt: string;
    status: AgentStatus;
}

interface ChangeRealTimeAgentPayload {
    serviceId: string;
    enable: boolean;
}

interface ChangeRealTimeAgentResponse { }

/**
 * @deprecated use getRunningSessions instead
 */
export const getRunningRealTimeAgents = async (): Promise<
    RunningRealTimeAgent[]
> => {
    const res =
        await api.get<ListRunningRealTimeAgentsResponse>('/realtime/agents');
    return res.data.agents;
};

/**
 * @deprecated use startSession/stopSession instead
 */
export const changeRealTimeAgent = async (
    payload: ChangeRealTimeAgentPayload
): Promise<ChangeRealTimeAgentResponse> => {
    const res = await api.post<ChangeRealTimeAgentResponse>(
        '/realtime/change',
        payload
    );
    return res.data;
};