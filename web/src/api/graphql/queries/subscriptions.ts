import { graphql } from '../graphql';

// ============================================================================
// Subscriptions
// ============================================================================

/**
 * Subscribe to job progress events
 * Can optionally filter by taskId and/or connectionId
 */
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

/**
 * Subscribe to transfer progress events (individual file transfers)
 * Can optionally filter by connectionId, taskId, and/or jobId
 */
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
