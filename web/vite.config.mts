import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { resolve } from "path";
import { defineConfig } from "vite";
import { polyfillsPlugin } from "./vite-plugin-polyfills";
import { visualizer } from "rollup-plugin-visualizer";

let devProxyServer = "http://localhost:28081";
if (process.env.DEV_PROXY_SERVER && process.env.DEV_PROXY_SERVER.length > 0) {
  console.log("Use devProxyServer from environment: ", process.env.DEV_PROXY_SERVER);
  devProxyServer = process.env.DEV_PROXY_SERVER;
}

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  return {
    plugins: [
      polyfillsPlugin(),
      react(),
      tailwindcss(),
      // Bundle analyzer for production builds (optional)
      mode === 'production' && visualizer({
        filename: './dist/stats.html',
        gzipSize: true,
        brotliSize: true,
      }),
    ].filter(Boolean),
    server: {
    host: "0.0.0.0",
    port: 25173,
    proxy: {
      "^/api": {
        target: devProxyServer,
        xfwd: true,
      },
      "^/memos.api.v1": {
        target: devProxyServer,
        xfwd: true,
      },
      "^/file": {
        target: devProxyServer,
        xfwd: true,
      },
    },
  },
  resolve: {
    alias: {
      "@/": `${resolve(__dirname, "src")}/`,
    },
  },
  build: {
    target: "es2020", // Modern browsers: Chrome 80+, Safari 13.1+, Firefox 72+
    // Remove console.log in production builds
    minify: "terser",
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true,
      },
    },
    rollupOptions: {
      output: {
        // Sanitize chunk names to avoid leading underscores (Go embed ignores them)
        chunkFileNames: "assets/[name]-[hash].js",
        entryFileNames: "assets/[name]-[hash].js",
        assetFileNames: "assets/[name]-[hash].[ext]",
        manualChunks(id) {
          // lodash-es internal modules - bundle into a single chunk
          if (id.includes("lodash-es") || id.includes("/_base")) {
            return "lodash-vendor";
          }
          // Merge core-js dependent packages into main entry to avoid polyfill issues
          if (id.includes("core-js/actual")) {
            return "polyfills";
          }
          if (id.includes("react") || id.includes("react-dom") || id.includes("react-router")) {
            return "react-vendor";
          }
          if (id.includes("@radix-ui") || id.includes("lucide-react")) {
            return "ui-vendor";
          }
          if (id.includes("react-markdown") || id.includes("remark-") || id.includes("rehype-") || id.includes("highlight.js")) {
            return "markdown-vendor";
          }
          if (id.includes("katex")) {
            return "math-vendor";
          }
          if (id.includes("@tanstack/react-query")) {
            return "query-vendor";
          }
          if (id.includes("i18next")) {
            return "i18n-vendor";
          }
          if (id.includes("mermaid")) {
            return "mermaid-vendor";
          }
          if (id.includes("leaflet")) {
            return "leaflet-vendor";
          }
        },
      },
    },
  },
  optimizeDeps: {
    include: ['core-js/actual', 'cytoscape', 'lodash-es', 'fuse.js', 'dayjs'],
  },
  };
});
