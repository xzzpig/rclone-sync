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
 * Query to list local directory contents
 */
export const FilesLocalQuery = graphql(
  `
    query FilesLocal($path: String!) {
      file {
        local(path: $path) {
          ...FileEntryFields
        }
      }
    }
  `,
  [FileEntryFragment]
);

/**
 * Query to list remote directory contents
 */
export const FilesRemoteQuery = graphql(
  `
    query FilesRemote($connectionId: ID!, $path: String!) {
      file {
        remote(connectionId: $connectionId, path: $path) {
          ...FileEntryFields
        }
      }
    }
  `,
  [FileEntryFragment]
);
