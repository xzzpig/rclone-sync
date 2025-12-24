import { initGraphQLTada } from 'gql.tada';
import type { introspection } from './graphql-env.d.ts';

export const graphql = initGraphQLTada<{
  introspection: introspection;
  scalars: {
    // 将 GraphQL Scalar 映射到 TypeScript 类型
    DateTime: string;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    JSON: any;
    BigInt: number;
    ID: string;
    StringMap: Record<string, string>;
  };
}>();

export type { FragmentOf, ResultOf, VariablesOf } from 'gql.tada';
export { readFragment } from 'gql.tada';
