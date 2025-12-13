import { Skeleton } from '@/components/ui/skeleton';
import { TableCell, TableRow } from '@/components/ui/table';
import { For } from 'solid-js';

interface TableSkeletonProps {
  rows?: number;
  columns: number;
  hiddenColumns?: number[];
}

const TableSkeleton = (props: TableSkeletonProps) => {
  const rows = () => props.rows ?? 5;

  return (
    <For each={Array(rows())}>
      {() => (
        <TableRow>
          <For each={Array(props.columns)}>
            {(_, index) => (
              <TableCell
                class={props.hiddenColumns?.includes(index()) ? 'hidden md:table-cell' : ''}
              >
                <Skeleton class="h-6 w-full" />
              </TableCell>
            )}
          </For>
        </TableRow>
      )}
    </For>
  );
};

export default TableSkeleton;
