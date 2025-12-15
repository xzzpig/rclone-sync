import { Toaster } from '@/components/ui/toast';
import AppShell from '@/layouts/AppShell';
import ConnectionLayout from '@/modules/connections/layouts/ConnectionLayout';
import History from '@/modules/connections/views/History';
import Log from '@/modules/connections/views/Log';
import Overview from '@/modules/connections/views/Overview';
import Settings from '@/modules/connections/views/Settings';
import Tasks from '@/modules/connections/views/Tasks';
import WelcomeView from '@/modules/core/views/WelcomeView';
import { HistoryProvider } from '@/store/history';
import { LocaleProvider } from '@/store/locale';
import { TaskProvider } from '@/store/tasks';
import { Route, Router } from '@solidjs/router';
import { Component } from 'solid-js';

const App: Component = () => {
  return (
    <LocaleProvider>
      <TaskProvider>
        <HistoryProvider>
          <Router root={AppShell}>
            <Route path="/" component={WelcomeView} />
            <Route path="/overview" component={WelcomeView} />
            <Route path="/connections/:connectionName" component={ConnectionLayout}>
              <Route path="/" component={Overview} />
              <Route path="/tasks" component={Tasks} />
              <Route path="/history" component={History} />
              <Route path="/log" component={Log} />
              <Route path="/settings" component={Settings} />
            </Route>
          </Router>
          <Toaster />
        </HistoryProvider>
      </TaskProvider>
    </LocaleProvider>
  );
};
export default App;
