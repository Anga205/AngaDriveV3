import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'
import tailwindcss from '@tailwindcss/vite'
import { viteSingleFile } from 'vite-plugin-singlefile';
import svg from 'vite-plugin-solid-svg'

export default defineConfig({
  plugins: [
    solid(),
    tailwindcss(),
    viteSingleFile(),
    svg()
  ],

  build: {
    target: "es2022",
    minify: "esbuild",
    cssMinify: true,
    sourcemap: false,
  },

  server: {
    port: 3000,
    allowedHosts: ['port3000.angadbhalla.com'],
  },

  resolve: {
    alias: {
      '@': '/src',
    }
  },
})