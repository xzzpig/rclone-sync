/**
 * GraphQL queries for File Browsing
 */
import { graphql } from '../graphql';

/**
 * File entry fields fragment
 */
const FileEntryFragment = graphql(`
  fragment FileEntryFields on FileEntry @_unmask {
    name
    path
    isDir
  }
`);

export type FileEntryFragmentType = typeof FileEntryFragment;

/**
 * Query to list directory contents (unified interface for local and remote)
 * @param connectionId - Optional connection ID. null = local filesystem
 * @param path - The path to browse
 * @param filters - Optional filter rules (rclone filter syntax)
 * @param includeFiles - Whether to include files (default: false, only directories)
 * @param basePath - Optional base path for filter matching
 */
export const FilesListQuery = graphql(
  `
    query FilesList(
      $connectionId: ID
      $path: String!
      $filters: [String!]
      $includeFiles: Boolean
      $basePath: String
    ) {
      file {
        list(
          connectionId: $connectionId
          path: $path
          filters: $filters
          includeFiles: $includeFiles
          basePath: $basePath
        ) {
          ...FileEntryFields
        }
      }
    }
  `,
  [FileEntryFragment]
);
