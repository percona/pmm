export enum DumpStatus {
  Unspecified = 'DUMP_STATUS_UNSPECIFIED',
  InProgress = 'DUMP_STATUS_IN_PROGRESS',
  Success = 'DUMP_STATUS_SUCCESS',
  Error = 'DUMP_STATUS_ERROR',
}

export interface Dump {
  dumpId: string;
  status: DumpStatus;
  serviceNames: string[];
  startTime: string;
  endTime: string;
  createdAt: string;
  encrypted: boolean;
}

export interface ListDumpsResponse {
  dumps: Dump[];
}

export interface StartDumpPayload {
  serviceNames: string[];
  startTime: string;
  endTime: string;
  exportQan: boolean;
  ignoreLoad: boolean;
  enableEncryption: boolean;
  encryptionPassword: string;
}

export interface StartDumpResponse {
  dumpId: string;
}

export interface DeleteDumpsPayload {
  dumpIds: string[];
}

export interface DumpLogChunk {
  chunkId: number;
  data: string;
}

export interface GetDumpLogsParams {
  dumpId: string;
  offset: number;
  limit: number;
}

export interface GetDumpLogsResponse {
  logs: DumpLogChunk[];
  end: boolean;
}

export interface SftpParameters {
  address: string;
  user: string;
  password: string;
  directory: string;
}

export interface UploadDumpsPayload {
  dumpIds: string[];
  sftpParameters: SftpParameters;
}
