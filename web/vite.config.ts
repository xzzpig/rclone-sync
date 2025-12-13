import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'
import Icons from 'unplugin-icons/vite'
import path from 'path'

export default defineConfig({
  plugins: [
    solid(),
    Icons({
      compiler: 'solid',
      autoInstall: false,
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  },
  build: {
    target: 'esnext',
    outDir: '../internal/ui/dist',
    emptyOutDir: true,
  },
})
