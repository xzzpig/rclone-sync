import { Provider } from '@urql/solid';
import type { JSX } from 'solid-js';
import { client } from './client';

interface GraphQLProviderProps {
  children: JSX.Element;
}

/**
 * GraphQL Provider component that wraps the application with urql client.
 * This enables GraphQL queries, mutations, and subscriptions throughout the app.
 */
export function GraphQLProvider(props: GraphQLProviderProps) {
  return <Provider value={client}>{props.children}</Provider>;
}

export default GraphQLProvider;
