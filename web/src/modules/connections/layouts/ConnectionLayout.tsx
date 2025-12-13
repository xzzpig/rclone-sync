import { Tabs, TabsIndicator, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useLocation, useNavigate } from '@solidjs/router';
import { ParentComponent } from 'solid-js';

const ConnectionLayout: ParentComponent = (props) => {
  const location = useLocation();
  const navigate = useNavigate();

  const currentTab = () => {
    const path = location.pathname;
    if (path.endsWith('/tasks')) return 'tasks';
    if (path.endsWith('/history')) return 'history';
    if (path.endsWith('/log')) return 'log';
    if (path.endsWith('/settings')) return 'settings';
    return 'overview';
  };

  return (
    <div class="flex min-h-0 flex-1 flex-col space-y-6">
      <header class="flex-none space-y-4">
        <Tabs
          value={currentTab()}
          onChange={(v) => navigate(v === 'overview' ? '.' : v)}
          class="w-full overflow-x-auto"
        >
          <TabsList>
            <TabsIndicator />
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="tasks">Tasks</TabsTrigger>
            <TabsTrigger value="history">History</TabsTrigger>
            <TabsTrigger value="log">Log</TabsTrigger>
            <TabsTrigger value="settings">Settings</TabsTrigger>
          </TabsList>
        </Tabs>
      </header>

      <div class="flex min-h-0 flex-1 flex-col overflow-hidden">{props.children}</div>
    </div>
  );
};

export default ConnectionLayout;
