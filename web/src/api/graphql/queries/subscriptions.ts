import { graphql } from '../graphql';

export const JOB_PROGRESS_SUBSCRIPTION = graphql(`
  subscription JobProgress($taskId: ID, $connectionId: ID) {
    jobProgress(taskId: $taskId, connectionId: $connectionId) {
      jobId
      taskId
      connectionId
      status
      filesTransferred
      bytesTransferred
      filesTotal
      bytesTotal
      filesDeleted
      errorCount
      startTime
      endTime
    }
  }
`);

export const TRANSFER_PROGRESS_SUBSCRIPTION = graphql(`
  subscription TransferProgress($connectionId: ID, $taskId: ID, $jobId: ID) {
    transferProgress(connectionId: $connectionId, taskId: $taskId, jobId: $jobId) {
      jobId
      taskId
      connectionId
      transfers {
        name
        size
        bytes
      }
    }
  }
`);

export const CACHE_STATUS_SUBSCRIPTION = graphql(`
  subscription CacheStatus($connectionId: ID!) {
    cacheStatus(connectionId: $connectionId) {
      connectionId
      running
      changeNotifySupported
      entriesCount
      dbSizeBytes
      lastNotifyTime
    }
  }
`);
