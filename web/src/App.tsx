import { Component } from 'solid-js';
import { Router, Route } from '@solidjs/router';
import Layout from './components/Layout';
import Remotes from './pages/Remotes';
import Tasks from './pages/Tasks';
import Dashboard from './pages/Dashboard';
import JobDetails from './pages/JobDetails';

const Settings: Component = () => (
  <div>
    <h1 class="text-2xl font-bold text-gray-800 mb-4">Settings</h1>
    <div class="bg-white p-6 rounded-lg shadow-sm">
      <p class="text-gray-600">Application settings.</p>
    </div>
  </div>
);

const App: Component = () => {
  return (
    <Router root={Layout}>
      <Route path="/" component={Dashboard} />
      <Route path="/remotes" component={Remotes} />
      <Route path="/tasks" component={Tasks} />
      <Route path="/jobs/:id" component={JobDetails} />
      <Route path="/settings" component={Settings} />
    </Router>
  );
};

export default App;
