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
      startTime
      endTime
    }
  }
`);
