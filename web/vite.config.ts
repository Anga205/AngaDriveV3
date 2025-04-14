import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'
import tailwindcss from '@tailwindcss/vite'
import { viteSingleFile } from 'vite-plugin-singlefile';

export default defineConfig({
  plugins: [
    solid(),
    tailwindcss(),
    viteSingleFile()
  ],
  server: {
    port: 3000,
    allowedHosts: ['port3000.angadbhalla.com'],
  }
})
