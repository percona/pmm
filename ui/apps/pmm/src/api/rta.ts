import {
  ListRunningSessionsResponse,
  RealtimeSession,
  RealtimeSessionStatus,
  SearchQueriesPayload,
  SearchQueriesResponse,
  StartSessionPayload,
  StartSessionResponse,
  StopSessionPayload,
} from 'types/rta.types';
import { api } from './api';
import { changeRealtimeAgent, getRunningRealtimeAgents } from './rta.fallback';
import { EmptyResponse } from 'types/util.types';

export const getRunningSessions = async (): Promise<RealtimeSession[]> => {
  try {
    const res = await api.get<ListRunningSessionsResponse>(
      '/realtimeanalytics/sessions'
    );
    return res.data.sessions;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    const agents = await getRunningRealtimeAgents();
    return agents.map<RealtimeSession>((agent) => ({
      status: RealtimeSessionStatus.unspecified,
      startTime: agent.startedAt,
      serviceId: agent.serviceId,
      serviceName: agent.serviceName,
      clusterName: agent.cluster,
    }));
  }
};

export const startSession = async (
  payload: StartSessionPayload
): Promise<StartSessionResponse> => {
  try {
    const res = await api.post<StartSessionResponse>(
      '/realtimeanalytics/sessions:start',
      payload
    );
    return res.data;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    await changeRealtimeAgent({
      serviceId: payload.serviceId,
      enable: true,
    });
    return {
      session: {
        serviceId: payload.serviceId,
        serviceName: '',
        clusterName: '',
        startTime: new Date().toISOString(),
        status: RealtimeSessionStatus.unspecified,
      },
    };
  }
};

export const stopSession = async (
  payload: StopSessionPayload
): Promise<EmptyResponse> => {
  try {
    const res = await api.post<EmptyResponse>(
      '/realtimeanalytics/sessions:stop',
      payload
    );
    return res.data;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    await changeRealtimeAgent({
      serviceId: payload.serviceId,
      enable: false,
    });
    return {};
  }
};

export const searchQueries = async (
  payload: SearchQueriesPayload
): Promise<SearchQueriesResponse> => {
  // todo: remove this once the API is implemented
  // const res = await api.post<SearchQueriesResponse>(
  //   '/realtimeanalytics/queries:search',
  //   payload
  // );
  // return res.data;

  return {
    queries: [
      {
        serviceId: '871bd114-2d1d-4228-bf28-e18313a82c26',
        serviceName: 'psmdb-1',
        queryId: '-1786893750',
        queryText:
          '{\n    "type": "op",\n    "host": "c4486b1ebd30:27017",\n    "desc": "conn14835",\n    "connectionId": 14835,\n    "client": "192.168.107.1:50012",\n    "clientMetadata": {\n        "driver": {\n            "name": "mongo-go-driver",\n            "version": "2.4.0"\n        },\n        "os": {\n            "type": "darwin",\n            "architecture": "arm64"\n        },\n        "platform": "go1.25.7"\n    },\n    "active": true,\n    "currentOpTime": "2026-02-12T10:22:05.351+00:00",\n    "effectiveUsers": [\n        {\n            "user": "root",\n            "db": "admin"\n        }\n    ],\n    "isFromUserConnection": true,\n    "threaded": true,\n    "opid": -1786893750,\n    "lsid": {\n        "id": {\n            "Subtype": 4,\n            "Data": "VFl8fpVMS3eaqnb0jEoMqQ=="\n        },\n        "uid": {\n            "Subtype": 0,\n            "Data": "Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg="\n        }\n    },\n    "op": "command",\n    "ns": "airline.$cmd",\n    "redacted": false,\n    "command": {\n        "insert": "flights",\n        "ordered": true,\n        "lsid": {\n            "id": {\n                "Subtype": 4,\n                "Data": "VFl8fpVMS3eaqnb0jEoMqQ=="\n            }\n        },\n        "$db": "airline"\n    },\n    "numYields": 0,\n    "queues": {\n        "ingress": {\n            "admissions": 1,\n            "totalTimeQueuedMicros": 0\n        },\n        "execution": {\n            "admissions": 0,\n            "totalTimeQueuedMicros": 0\n        }\n    },\n    "currentQueue": null,\n    "locks": {},\n    "waitingForLock": false,\n    "lockStats": {},\n    "waitingForFlowControl": false,\n    "flowControlStats": {}\n}',
        queryRawJson:
          '{\n    "type": "op",\n    "host": "c4486b1ebd30:27017",\n    "desc": "conn14835",\n    "connectionId": 14835,\n    "client": "192.168.107.1:50012",\n    "clientMetadata": {\n        "driver": {\n            "name": "mongo-go-driver",\n            "version": "2.4.0"\n        },\n        "os": {\n            "type": "darwin",\n            "architecture": "arm64"\n        },\n        "platform": "go1.25.7"\n    },\n    "active": true,\n    "currentOpTime": "2026-02-12T10:22:05.351+00:00",\n    "effectiveUsers": [\n        {\n            "user": "root",\n            "db": "admin"\n        }\n    ],\n    "isFromUserConnection": true,\n    "threaded": true,\n    "opid": -1786893750,\n    "lsid": {\n        "id": {\n            "Subtype": 4,\n            "Data": "VFl8fpVMS3eaqnb0jEoMqQ=="\n        },\n        "uid": {\n            "Subtype": 0,\n            "Data": "Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg="\n        }\n    },\n    "op": "command",\n    "ns": "airline.$cmd",\n    "redacted": false,\n    "command": {\n        "insert": "flights",\n        "ordered": true,\n        "lsid": {\n            "id": {\n                "Subtype": 4,\n                "Data": "VFl8fpVMS3eaqnb0jEoMqQ=="\n            }\n        },\n        "$db": "airline"\n    },\n    "numYields": 0,\n    "queues": {\n        "ingress": {\n            "admissions": 1,\n            "totalTimeQueuedMicros": 0\n        },\n        "execution": {\n            "admissions": 0,\n            "totalTimeQueuedMicros": 0\n        }\n    },\n    "currentQueue": null,\n    "locks": {},\n    "waitingForLock": false,\n    "lockStats": {},\n    "waitingForFlowControl": false,\n    "flowControlStats": {}\n}',
        queryExecutionDuration: null,
        queryCollectTime: '2026-02-12T10:22:05.352335388Z',
        clientAddress: '192.168.107.1:50012',
        mongoDbPayload: {
          dbInstanceAddress: 'c4486b1ebd30:27017',
          clientAppName: '',
          databaseName: 'airline',
          operation: 'command',
          operationStartTime: '2026-02-12T10:22:05.351Z',
          username: 'root',
          planSummary: '',
        },
      },
      {
        serviceId: '871bd114-2d1d-4228-bf28-e18313a82c26',
        serviceName: 'psmdb-1',
        queryId: '-1787103621',
        queryText: 'db.flights.insert(?, {"ordered":true})',
        queryRawJson:
          '{\n    "type": "op",\n    "host": "c4486b1ebd30:27017",\n    "desc": "conn14838",\n    "connectionId": 14838,\n    "client": "192.168.107.1:50036",\n    "clientMetadata": {\n        "driver": {\n            "name": "mongo-go-driver",\n            "version": "2.4.0"\n        },\n        "os": {\n            "type": "darwin",\n            "architecture": "arm64"\n        },\n        "platform": "go1.25.7"\n    },\n    "active": true,\n    "currentOpTime": "2026-02-12T10:22:05.351+00:00",\n    "effectiveUsers": [\n        {\n            "user": "root",\n            "db": "admin"\n        }\n    ],\n    "isFromUserConnection": true,\n    "threaded": true,\n    "opid": -1787103621,\n    "lsid": {\n        "id": {\n            "Subtype": 4,\n            "Data": "JpmdZc2IStONtpTpwRLhLw=="\n        },\n        "uid": {\n            "Subtype": 0,\n            "Data": "Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg="\n        }\n    },\n    "secs_running": 0,\n    "microsecs_running": 16,\n    "op": "insert",\n    "ns": "airline.flights",\n    "redacted": false,\n    "command": {\n        "insert": "flights",\n        "ordered": true,\n        "lsid": {\n            "id": {\n                "Subtype": 4,\n                "Data": "JpmdZc2IStONtpTpwRLhLw=="\n            }\n        },\n        "$db": "airline"\n    },\n    "numYields": 0,\n    "queues": {\n        "ingress": {\n            "admissions": 1,\n            "totalTimeQueuedMicros": 0\n        },\n        "execution": {\n            "admissions": 1,\n            "totalTimeQueuedMicros": 0\n        }\n    },\n    "currentQueue": null,\n    "locks": {\n        "ReplicationStateTransition": "w",\n        "Global": "w",\n        "Database": "w",\n        "Collection": "w"\n    },\n    "waitingForLock": false,\n    "lockStats": {\n        "ReplicationStateTransition": {\n            "acquireCount": {\n                "w": 1\n            }\n        },\n        "Global": {\n            "acquireCount": {\n                "w": 1\n            }\n        },\n        "Database": {\n            "acquireCount": {\n                "w": 1\n            }\n        },\n        "Collection": {\n            "acquireCount": {\n                "w": 1\n            }\n        }\n    },\n    "waitingForFlowControl": false,\n    "flowControlStats": {\n        "acquireCount": 1\n    }\n}',
        queryExecutionDuration: '0.000016s',
        queryCollectTime: '2026-02-12T10:22:05.352335388Z',
        clientAddress: '192.168.107.1:50036',
        mongoDbPayload: {
          dbInstanceAddress: 'c4486b1ebd30:27017',
          clientAppName: '',
          databaseName: 'airline',
          collection: 'flights',
          operation: 'insert',
          operationStartTime: '2026-02-12T10:22:05.351Z',
          username: 'root',
          planSummary: '',
        },
      },
      {
        serviceId: '871bd114-2d1d-4228-bf28-e18313a82c26',
        serviceName: 'psmdb-1',
        queryId: '1578052890',
        queryText: 'admin.$cmd(?)',
        queryRawJson:
          '{\n    "type": "op",\n    "host": "c4486b1ebd30:27017",\n    "desc": "conn14601",\n    "connectionId": 14601,\n    "client": "127.0.0.1:42372",\n    "appName": "mongosh 2.5.10",\n    "clientMetadata": {\n        "application": {\n            "name": "mongosh 2.5.10"\n        },\n        "driver": {\n            "name": "nodejs|mongosh",\n            "version": "6.19.0|2.5.10"\n        },\n        "platform": "Node.js v20.19.6, LE",\n        "os": {\n            "name": "linux",\n            "architecture": "arm64",\n            "version": "6.1.0-41-arm64",\n            "type": "Linux"\n        },\n        "env": {\n            "container": {\n                "runtime": "docker"\n            }\n        }\n    },\n    "active": true,\n    "currentOpTime": "2026-02-12T10:22:05.351+00:00",\n    "isFromUserConnection": true,\n    "threaded": true,\n    "opid": 1578052890,\n    "secs_running": 8,\n    "microsecs_running": 8690622,\n    "op": "command",\n    "ns": "admin.$cmd",\n    "redacted": false,\n    "command": {\n        "hello": 1,\n        "maxAwaitTimeMS": 10000,\n        "topologyVersion": {\n            "processId": "6980678dcfe9e64b7c5ab4a1",\n            "counter": 0\n        },\n        "$db": "admin"\n    },\n    "numYields": 0,\n    "queues": {\n        "ingress": {\n            "admissions": 1,\n            "totalTimeQueuedMicros": 0\n        },\n        "execution": {\n            "admissions": 0,\n            "totalTimeQueuedMicros": 0\n        }\n    },\n    "currentQueue": null,\n    "locks": {},\n    "waitingForLock": false,\n    "lockStats": {},\n    "waitingForFlowControl": false,\n    "flowControlStats": {}\n}',
        queryExecutionDuration: '8.690622s',
        queryCollectTime: '2026-02-12T10:22:05.352335388Z',
        clientAddress: '127.0.0.1:42372',
        mongoDbPayload: {
          dbInstanceAddress: 'c4486b1ebd30:27017',
          clientAppName: 'mongosh 2.5.10',
          databaseName: 'admin',
          operation: 'command',
          operationStartTime: '2026-02-12T10:22:05.351Z',
          username: '',
          planSummary: '',
        },
      },
      {
        serviceId: '871bd114-2d1d-4228-bf28-e18313a82c26',
        serviceName: 'psmdb-1',
        queryId: '-1858348030',
        queryText: 'admin.$cmd(?)',
        queryRawJson:
          '{\n    "type": "op",\n    "host": "c4486b1ebd30:27017",\n    "desc": "conn14828",\n    "connectionId": 14828,\n    "client": "192.168.107.1:49954",\n    "clientMetadata": {\n        "driver": {\n            "name": "mongo-go-driver",\n            "version": "2.4.0"\n        },\n        "os": {\n            "type": "darwin",\n            "architecture": "arm64"\n        },\n        "platform": "go1.25.7"\n    },\n    "active": true,\n    "currentOpTime": "2026-02-12T10:22:05.351+00:00",\n    "isFromUserConnection": true,\n    "threaded": true,\n    "opid": -1858348030,\n    "secs_running": 1,\n    "microsecs_running": 1790303,\n    "op": "command",\n    "ns": "admin.$cmd",\n    "redacted": false,\n    "command": {\n        "hello": 1,\n        "helloOk": true,\n        "topologyVersion": {\n            "processId": "6980678dcfe9e64b7c5ab4a1",\n            "counter": 0\n        },\n        "maxAwaitTimeMS": 10000,\n        "$db": "admin"\n    },\n    "numYields": 0,\n    "queues": {\n        "ingress": {\n            "admissions": 1,\n            "totalTimeQueuedMicros": 0\n        },\n        "execution": {\n            "admissions": 0,\n            "totalTimeQueuedMicros": 0\n        }\n    },\n    "currentQueue": null,\n    "locks": {},\n    "waitingForLock": false,\n    "lockStats": {},\n    "waitingForFlowControl": false,\n    "flowControlStats": {}\n}',
        queryExecutionDuration: '1.790303s',
        queryCollectTime: '2026-02-12T10:22:05.352335388Z',
        clientAddress: '192.168.107.1:49954',
        mongoDbPayload: {
          dbInstanceAddress: 'c4486b1ebd30:27017',
          clientAppName: '',
          databaseName: 'admin',
          collection: '',
          operation: 'command',
          operationStartTime: '2026-02-12T10:22:05.351Z',
          username: '',
          planSummary: '',
        },
      },
      {
        serviceId: '871bd114-2d1d-4228-bf28-e18313a82c26',
        serviceName: 'psmdb-1',
        queryId: '1625924839',
        queryText: 'admin.$cmd(?)',
        queryRawJson:
          '{\n    "type": "op",\n    "host": "c4486b1ebd30:27017",\n    "desc": "conn14809",\n    "connectionId": 14809,\n    "client": "192.168.107.1:16702",\n    "appName": "DataGrip",\n    "clientMetadata": {\n        "application": {\n            "name": "DataGrip"\n        },\n        "driver": {\n            "name": "mongo-java-driver|sync",\n            "version": "4.11.1"\n        },\n        "os": {\n            "type": "Darwin",\n            "name": "Mac OS X",\n            "architecture": "aarch64",\n            "version": "26.2"\n        },\n        "platform": "Java/JetBrains s.r.o./21.0.9+10-b1163.86"\n    },\n    "active": true,\n    "currentOpTime": "2026-02-12T10:22:05.351+00:00",\n    "isFromUserConnection": true,\n    "threaded": true,\n    "opid": 1625924839,\n    "secs_running": 7,\n    "microsecs_running": 7829116,\n    "op": "command",\n    "ns": "admin.$cmd",\n    "redacted": false,\n    "command": {\n        "hello": 1,\n        "helloOk": true,\n        "topologyVersion": {\n            "processId": "6980678dcfe9e64b7c5ab4a1",\n            "counter": 0\n        },\n        "maxAwaitTimeMS": 10000,\n        "$db": "admin",\n        "$readPreference": {\n            "mode": "primaryPreferred"\n        }\n    },\n    "numYields": 0,\n    "queues": {\n        "ingress": {\n            "admissions": 1,\n            "totalTimeQueuedMicros": 0\n        },\n        "execution": {\n            "admissions": 0,\n            "totalTimeQueuedMicros": 0\n        }\n    },\n    "currentQueue": null,\n    "locks": {},\n    "waitingForLock": false,\n    "lockStats": {},\n    "waitingForFlowControl": false,\n    "flowControlStats": {}\n}',
        queryExecutionDuration: '7.829116s',
        queryCollectTime: '2026-02-12T10:22:05.352335388Z',
        clientAddress: '192.168.107.1:16702',
        mongoDbPayload: {
          dbInstanceAddress: 'c4486b1ebd30:27017',
          clientAppName: 'DataGrip',
          databaseName: 'admin',
          collection: '',
          operation: 'command',
          operationStartTime: '2026-02-12T10:22:05.351Z',
          username: '',
          planSummary: '',
        },
      },
    ],
  };
};
