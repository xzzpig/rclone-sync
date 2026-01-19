/**
 * GraphQL queries for Hooks
 */
import { graphql } from '../graphql';

/**
 * Query to list hooks by task or connection ID
 */
export const HooksListQuery = graphql(`
  query HooksList($taskId: ID, $connectionId: ID, $event: HookEvent) {
    hook {
      list(taskId: $taskId, connectionId: $connectionId, event: $event) {
        id
        enabled
        priority
        event
        type
        onError
        config {
          url
          method
          headers
          body
          command
          workDir
          timeout
        }
        createdAt
        updatedAt
      }
    }
  }
`);

/**
 * Query to get a single hook by ID
 */
export const HookGetQuery = graphql(`
  query HookGet($id: ID!) {
    hook {
      get(id: $id) {
        id
        enabled
        priority
        event
        type
        onError
        config {
          url
          method
          headers
          body
          command
          workDir
          timeout
        }
        task {
          id
          name
        }
        connection {
          id
          name
        }
        createdAt
        updatedAt
      }
    }
  }
`);

/**
 * Mutation to create a new hook
 */
export const HookCreateMutation = graphql(`
  mutation HookCreate($taskId: ID, $connectionId: ID, $input: HookInput!) {
    hook {
      create(taskId: $taskId, connectionId: $connectionId, input: $input) {
        id
        enabled
        priority
        event
        type
        onError
        config {
          url
          method
          headers
          body
          command
          workDir
          timeout
        }
        createdAt
        updatedAt
      }
    }
  }
`);

/**
 * Mutation to update an existing hook
 */
export const HookUpdateMutation = graphql(`
  mutation HookUpdate($id: ID!, $input: UpdateHookInput!) {
    hook {
      update(id: $id, input: $input) {
        id
        enabled
        priority
        event
        type
        onError
        config {
          url
          method
          headers
          body
          command
          workDir
          timeout
        }
        createdAt
        updatedAt
      }
    }
  }
`);

/**
 * Mutation to delete a hook
 */
export const HookDeleteMutation = graphql(`
  mutation HookDelete($id: ID!) {
    hook {
      delete(id: $id) {
        id
      }
    }
  }
`);
