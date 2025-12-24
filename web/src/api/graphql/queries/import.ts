import { graphql } from '../graphql';

// ============================================================================
// Mutations
// ============================================================================

/**
 * Parse rclone configuration content
 */
export const IMPORT_PARSE = graphql(`
  mutation ImportParse($input: ImportParseInput!) {
    import {
      parse(input: $input) {
        ... on ImportParseSuccess {
          __typename
          connections {
            name
            type
            config
          }
        }
        ... on ImportParseError {
          __typename
          error
          line
        }
      }
    }
  }
`);

/**
 * Execute import of connections
 */
export const IMPORT_EXECUTE = graphql(`
  mutation ImportExecute($input: ImportExecuteInput!) {
    import {
      execute(input: $input) {
        connections {
          id
          name
          type
          createdAt
        }
        skippedCount
      }
    }
  }
`);
