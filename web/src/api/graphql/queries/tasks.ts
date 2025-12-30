/**
 * GraphQL queries for Tasks
 */
import { graphql } from '../graphql';

/**
 * Query to list all tasks with pagination
 */
export const TasksListQuery = graphql(`
  query TasksList($pagination: PaginationInput) {
    task {
      list(pagination: $pagination) {
        items {
          id
          name
          sourcePath
          remotePath
          direction
          schedule
          realtime
          createdAt
          updatedAt
          connection {
            id
            name
            type
          }
          options {
            conflictResolution
            filters
            noDelete
            transfers
          }
          latestJob {
            id
            status
            startTime
            endTime
            filesTransferred
            bytesTransferred
          }
        }
      }
    }
  }
`);

/**
 * Query to get a single task by ID
 */
export const TaskGetQuery = graphql(`
  query TaskGet($id: ID!) {
    task {
      get(id: $id) {
        id
        name
        sourcePath
        remotePath
        direction
        schedule
        realtime
        createdAt
        updatedAt
        connection {
          id
          name
          type
        }
        options {
          conflictResolution
          filters
          noDelete
          transfers
        }
        latestJob {
          id
          status
          startTime
          endTime
        }
      }
    }
  }
`);

/**
 * Mutation to create a new task
 */
export const TaskCreateMutation = graphql(`
  mutation TaskCreate($input: CreateTaskInput!) {
    task {
      create(input: $input) {
        id
        name
        sourcePath
        remotePath
        direction
        schedule
        realtime
        createdAt
        updatedAt
      }
    }
  }
`);

/**
 * Mutation to update an existing task
 */
export const TaskUpdateMutation = graphql(`
  mutation TaskUpdate($id: ID!, $input: UpdateTaskInput!) {
    task {
      update(id: $id, input: $input) {
        id
        name
        sourcePath
        remotePath
        direction
        schedule
        realtime
        createdAt
        updatedAt
      }
    }
  }
`);

/**
 * Mutation to delete a task
 */
export const TaskDeleteMutation = graphql(`
  mutation TaskDelete($id: ID!) {
    task {
      delete(id: $id) {
        id
        name
      }
    }
  }
`);

/**
 * Mutation to run a task (start a sync job)
 */
export const TaskRunMutation = graphql(`
  mutation TaskRun($taskId: ID!) {
    task {
      run(taskId: $taskId) {
        id
        status
        startTime
      }
    }
  }
`);
