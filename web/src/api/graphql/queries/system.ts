/**
 * GraphQL queries for System Configuration
 */
import { graphql } from '../graphql';

/**
 * Query to get application configuration
 */
export const AppConfigQuery = graphql(`
  query AppConfig {
    appConfig {
      hook {
        enabled
      }
    }
  }
`);
