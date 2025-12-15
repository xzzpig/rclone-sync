import * as m from '@/paraglide/messages.js';
import { useIsMobile } from '@/lib/media-query';
import MobileHeader from '@/modules/core/components/MobileHeader';
import Sidebar from '@/modules/core/components/Sidebar';
import { TaskProvider, useTasks } from '@/store/tasks';
import { useLocation } from '@solidjs/router';
import { ParentComponent, Show, onMount } from 'solid-js';

// Inner component that uses the TaskProvider context
const AppShellInner: ParentComponent = (props) => {
  const [, actions] = useTasks();

  // Load all tasks on mount for sidebar status indicators
  // and start global SSE subscription for real-time updates
  onMount(() => {
    actions.loadTasks();
    actions.startGlobalSseSubscription();
  });

  return <>{props.children}</>;
};

const AppShell: ParentComponent = (props) => {
  const isMobile = useIsMobile();
  const location = useLocation();

  // In mobile view (Stack Navigation):
  // - If we are at root path '/', show Sidebar (Connection List)
  // - If we are at any other path, show Main Content
  const isRootPath = () => location.pathname === '/' || location.pathname === '';
  const showSidebar = () => !isMobile() || isRootPath();
  const showContent = () => !isMobile() || !isRootPath();

  const getPageTitle = () => {
    const parts = location.pathname.split('/').filter(Boolean);
    // Expected pattern: parts[0] is 'connections', parts[1] is name, parts[2] is subpage
    if (parts[0] === 'overview') {
      return m.common_overview();
    }
    if (parts[0] === 'connections' && parts[1]) {
      // Connection Overview: "my-nas"
      return decodeURIComponent(parts[1]);
    }
    return 'Cloud Sync';
  };

  return (
    <TaskProvider>
      <AppShellInner>
        <div class="flex h-screen min-h-0 w-full bg-background text-foreground">
          <Show when={showSidebar()}>
            <aside
              class={`${isMobile() ? 'w-full' : 'w-[280px]'} z-20 h-full shrink-0 border-r border-border bg-card`}
            >
              <Sidebar />
            </aside>
          </Show>

          <Show when={showContent()}>
            <main class="relative flex h-full min-h-0 min-w-0 flex-1 flex-col bg-muted/30">
              <Show when={isMobile()}>
                <MobileHeader title={getPageTitle()} showBack={true} />
              </Show>
              <div class="flex h-full min-h-0 flex-1 flex-col">
                <div class="container mx-auto flex min-h-0 max-w-7xl flex-1 flex-col px-4 py-6">
                  {props.children}
                </div>
              </div>
            </main>
          </Show>
        </div>
      </AppShellInner>
    </TaskProvider>
  );
};

export default AppShell;
