import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: true, // This makes the server listen on all network interfaces
    port: 5173, // You can explicitly set the port here if you want
    // Add this proxy configuration
    proxy: {
      // Proxy requests from /api to the backend server
      '/api': {
        target: 'http://localhost:8080', // The Go backend server
        changeOrigin: true, // Needed for virtual hosted sites
        secure: false,      // If you are not using https
      },
    },
  },
  // build: {
  //   outDir: 'dist',
  //   reportCompressedSize: true,
  //   commonjsOptions: {
  //     transformMixedEsModules: true,
  //   },
  // },
})
