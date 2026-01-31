import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { resolve } from "path";
import { defineConfig } from "vite";

let devProxyServer = "http://localhost:28081";
if (process.env.DEV_PROXY_SERVER && process.env.DEV_PROXY_SERVER.length > 0) {
  console.log("Use devProxyServer from environment: ", process.env.DEV_PROXY_SERVER);
  devProxyServer = process.env.DEV_PROXY_SERVER;
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
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
    rollupOptions: {
      output: {
        manualChunks: {
          "react-vendor": ["react", "react-dom", "react-router-dom"],
          "ui-vendor": [
            "@radix-ui/react-dialog",
            "@radix-ui/react-dropdown-menu",
            "@radix-ui/react-popover",
            "@radix-ui/react-select",
            "@radix-ui/react-tooltip",
            "@radix-ui/react-checkbox",
            "@radix-ui/react-switch",
            "lucide-react",
          ],
          "markdown-vendor": ["react-markdown", "remark-gfm", "remark-breaks", "rehype-raw", "rehype-sanitize", "highlight.js"],
          "math-vendor": ["katex", "rehype-katex", "remark-math"],
          "query-vendor": ["@tanstack/react-query"],
          "i18n-vendor": ["i18next", "react-i18next"],
          "graph-vendor": ["cytoscape", "react-force-graph-2d"],
          "utils-vendor": ["dayjs", "lodash-es", "fuse.js", "uuid"],
          "mermaid-vendor": ["mermaid"],
          "leaflet-vendor": ["leaflet", "react-leaflet"],
        },
      },
    },
  },
});
