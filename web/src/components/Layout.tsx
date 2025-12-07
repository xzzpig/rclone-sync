import { Component, JSX } from 'solid-js';
import { A, useLocation } from '@solidjs/router';
import { FiHome, FiCloud, FiActivity, FiSettings, FiMenu } from 'solid-icons/fi';
import { createSignal } from 'solid-js';
import clsx from 'clsx';

const SidebarItem: Component<{ href: string; icon: Component<{ class?: string }>; label: string }> = (props) => {
  const location = useLocation();
  const active = () => location.pathname === props.href;

  return (
    <A
      href={props.href}
      class={clsx(
        'flex items-center px-6 py-3 text-gray-600 hover:bg-gray-100 hover:text-gray-900 transition-colors',
        active() && 'bg-blue-50 text-blue-600 border-r-4 border-blue-600'
      )}
    >
      <props.icon class="w-5 h-5 mr-3" />
      <span class="font-medium">{props.label}</span>
    </A>
  );
};

const Layout: Component<any> = (props) => {
  const [sidebarOpen, setSidebarOpen] = createSignal(true);

  return (
    <div class="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <aside
        class={clsx(
          'bg-white shadow-md transition-all duration-300 ease-in-out flex flex-col',
          sidebarOpen() ? 'w-64' : 'w-0 overflow-hidden'
        )}
      >
        <div class="h-16 flex items-center justify-center border-b px-6">
          <FiCloud class="w-8 h-8 text-blue-600 mr-2" />
          <span class="text-xl font-bold text-gray-800">Cloud Sync</span>
        </div>

        <nav class="flex-1 py-6 overflow-y-auto">
          <SidebarItem href="/" icon={FiHome} label="Dashboard" />
          <SidebarItem href="/remotes" icon={FiCloud} label="Remotes" />
          <SidebarItem href="/tasks" icon={FiActivity} label="Sync Tasks" />
          <SidebarItem href="/settings" icon={FiSettings} label="Settings" />
        </nav>

        <div class="p-4 border-t text-xs text-gray-400 text-center">
          v0.1.0
        </div>
      </aside>

      {/* Main Content */}
      <div class="flex-1 flex flex-col overflow-hidden">
        <header class="h-16 bg-white shadow-sm flex items-center px-6 justify-between z-10">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen())}
            class="p-2 rounded-md hover:bg-gray-100 text-gray-600 focus:outline-none"
          >
            <FiMenu class="w-6 h-6" />
          </button>
          <div class="flex items-center space-x-4">
            {/* Add user profile or other header items here */}
            <div class="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center text-blue-600 font-bold">
              A
            </div>
          </div>
        </header>

        <main class="flex-1 overflow-x-hidden overflow-y-auto bg-gray-50 p-6">
          {props.children}
        </main>
      </div>
    </div>
  );
};

export default Layout;
