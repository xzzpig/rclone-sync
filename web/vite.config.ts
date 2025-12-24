import { paraglideVitePlugin as paraglide } from '@inlang/paraglide-js';
import { defineConfig } from 'vite';
import solid from 'vite-plugin-solid';
import solidSvg from 'vite-plugin-solid-svg';
import Icons from 'unplugin-icons/vite';
import path from 'path';

export default defineConfig({
  publicDir: 'src/public',
  plugins: [
    paraglide({ project: './project.inlang', outdir: './src/paraglide' }),
    solid(),
    solidSvg(),
    Icons({
      compiler: 'solid',
      autoInstall: false,
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true, // Enable WebSocket proxying for GraphQL subscriptions
      },
    },
  },
  build: {
    target: 'esnext',
    outDir: '../internal/ui/dist',
    emptyOutDir: true,
  },
});
