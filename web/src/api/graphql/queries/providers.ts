/**
 * GraphQL queries for Providers
 */
import { graphql } from '../graphql';

/**
 * Query to list all available storage providers
 */
export const ProvidersListQuery = graphql(`
  query ProvidersList {
    provider {
      list {
        name
        description
        prefix
      }
    }
  }
`);

/**
 * Query to get a single provider with its configuration options
 */
export const ProviderGetQuery = graphql(`
  query ProviderGet($name: String!) {
    provider {
      get(name: $name) {
        name
        description
        prefix
        options {
          name
          help
          required
          isPassword
          default
          examples {
            value
            help
          }
          advanced
          groups
          exclusive
          type
        }
      }
    }
  }
`);
