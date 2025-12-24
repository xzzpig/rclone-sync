import StatusIcon from '@/components/common/StatusIcon';
import type { StatusType } from '@/lib/types';
import { A } from '@solidjs/router';
import { Component } from 'solid-js';
import IconHardDrive from '~icons/lucide/hard-drive';

// Connection type for sidebar display
interface ConnectionInfo {
  id: string;
  name: string;
  type: string;
}

interface ConnectionSidebarItemProps {
  connection: ConnectionInfo;
  status: StatusType;
}

export const ConnectionSidebarItem: Component<ConnectionSidebarItemProps> = (props) => {
  return (
    <A
      href={`/connections/${props.connection.id}`}
      class="group flex w-full items-center rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
      activeClass="bg-secondary text-foreground shadow-sm"
    >
      <IconHardDrive class="mr-3 size-4 shrink-0 opacity-70" />
      <div class="flex min-w-0 flex-1 flex-col items-start">
        <span class="w-full truncate text-left">{props.connection.name}</span>
        <span class="w-full truncate text-left text-[10px] font-normal text-muted-foreground/70">
          {props.connection.type}
        </span>
      </div>
      <span class="ml-auto shrink-0 pl-2">
        <StatusIcon status={props.status} class="size-4" />
      </span>
    </A>
  );
};
