/**
 * GraphQL queries for Connections
 */
import { graphql } from '../graphql';

/**
 * Query to list all connections with pagination
 */
export const ConnectionsListQuery = graphql(`
  query ConnectionsList($pagination: PaginationInput) {
    connection {
      list(pagination: $pagination) {
        items {
          id
          name
          type
        }
      }
    }
  }
`);

/**
 * Query to get basic connection info (for display purposes)
 */
export const ConnectionGetBasicQuery = graphql(`
  query ConnectionGetBasic($id: ID!) {
    connection {
      get(id: $id) {
        id
        name
        type
      }
    }
  }
`);

/**
 * Query to get connection with config (for settings/editing)
 */
export const ConnectionGetConfigQuery = graphql(`
  query ConnectionGetConfig($id: ID!) {
    connection {
      get(id: $id) {
        id
        name
        type
        config
      }
    }
  }
`);

/**
 * Query to get connection with quota (for overview/storage display)
 * Includes extended quota fields: trashed, other, objects
 */
export const ConnectionGetQuotaQuery = graphql(`
  query ConnectionGetQuota($id: ID!) {
    connection {
      get(id: $id) {
        id
        name
        quota {
          total
          used
          trashed
          other
          objects
        }
      }
    }
  }
`);

/**
 * Mutation to create a new connection
 */
export const ConnectionCreateMutation = graphql(`
  mutation ConnectionCreate($input: CreateConnectionInput!) {
    connection {
      create(input: $input) {
        id
        name
        type
        createdAt
        updatedAt
      }
    }
  }
`);

/**
 * Mutation to update an existing connection
 */
export const ConnectionUpdateMutation = graphql(`
  mutation ConnectionUpdate($id: ID!, $input: UpdateConnectionInput!) {
    connection {
      update(id: $id, input: $input) {
        id
        name
        type
        createdAt
        updatedAt
      }
    }
  }
`);

/**
 * Mutation to delete a connection
 */
export const ConnectionDeleteMutation = graphql(`
  mutation ConnectionDelete($id: ID!) {
    connection {
      delete(id: $id) {
        id
        name
      }
    }
  }
`);

/**
 * Mutation to test a saved connection
 */
export const ConnectionTestMutation = graphql(`
  mutation ConnectionTest($id: ID!) {
    connection {
      test(id: $id) {
        __typename
        ... on ConnectionTestSuccess {
          message
        }
        ... on ConnectionTestFailure {
          error
        }
      }
    }
  }
`);

/**
 * Mutation to test an unsaved connection configuration
 */
export const ConnectionTestUnsavedMutation = graphql(`
  mutation ConnectionTestUnsaved($input: TestConnectionInput!) {
    connection {
      testUnsaved(input: $input) {
        __typename
        ... on ConnectionTestSuccess {
          message
        }
        ... on ConnectionTestFailure {
          error
        }
      }
    }
  }
`);
