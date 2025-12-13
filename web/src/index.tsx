/* @refresh reload */
import { ColorModeProvider, ColorModeScript } from '@kobalte/core';
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query';
import { render } from 'solid-js/web';
import App from './App.tsx';
import './index.css';

const queryClient = new QueryClient();
const root = document.getElementById('root');

render(
  () => (
    <QueryClientProvider client={queryClient}>
      <ColorModeScript />
      <ColorModeProvider>
        <App />
      </ColorModeProvider>
    </QueryClientProvider>
  ),
  root!
);
