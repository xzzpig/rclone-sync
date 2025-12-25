/**
 * GraphQL queries for Jobs and Logs
 */
import { graphql } from '../graphql';

/**
 * Query to list jobs with optional task and connection filter
 */
export const JobsListQuery = graphql(`
  query JobsList(
    $taskId: ID
    $connectionId: ID
    $pagination: PaginationInput
    $withConnection: Boolean! = false
    $withProgress: Boolean! = false
  ) {
    job {
      list(taskId: $taskId, connectionId: $connectionId, pagination: $pagination) {
        items {
          id
          status
          trigger
          startTime
          endTime
          filesTransferred
          bytesTransferred
          filesDeleted
          errorCount
          task {
            id
            name
            connection @include(if: $withConnection) {
              id
              name
            }
          }
          progress @include(if: $withProgress) {
            filesTransferred
            bytesTransferred
            filesTotal
            bytesTotal
            filesDeleted
            errorCount
          }
        }
        totalCount
        pageInfo {
          limit
          offset
          hasNextPage
          hasPreviousPage
        }
      }
    }
  }
`);

/**
 * Query to get job progress (for running jobs)
 */
export const JobProgressQuery = graphql(`
  query JobProgress($id: ID!) {
    job {
      progress(id: $id) {
        jobId
        taskId
        connectionId
        status
        filesTransferred
        filesTotal
        bytesTransferred
        bytesTotal
        startTime
        endTime
      }
    }
  }
`);

/**
 * Query to list logs with flexible filtering (matches original REST API)
 */
export const LogsListQuery = graphql(`
  query LogsList(
    $connectionId: ID!
    $taskId: ID
    $jobId: ID
    $level: LogLevel
    $pagination: PaginationInput
  ) {
    log {
      list(
        connectionId: $connectionId
        taskId: $taskId
        jobId: $jobId
        level: $level
        pagination: $pagination
      ) {
        items {
          id
          level
          time
          path
          what
          size
        }
        totalCount
        pageInfo {
          limit
          offset
          hasNextPage
          hasPreviousPage
        }
      }
    }
  }
`);
