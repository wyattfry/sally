import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

const DEFAULT_BACKEND_BASE_URL = "http://10.0.0.104:8080";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const backendBaseUrl = env.VITE_SALLY_BACKEND_BASE_URL || DEFAULT_BACKEND_BASE_URL;
  const backendOrigin = new URL(backendBaseUrl).origin;

  return {
    plugins: [
      react(),
      {
        name: "sally-manifest-backend-origin",
        writeBundle() {
          const manifestPath = resolve(__dirname, "dist/manifest.json");
          const manifest = readFileSync(manifestPath, "utf8").replaceAll(
            "__SALLY_BACKEND_ORIGIN__",
            backendOrigin
          );
          writeFileSync(manifestPath, manifest);
        }
      }
    ],
    test: {
      environment: "jsdom",
      globals: true,
      environmentOptions: {
        jsdom: {
          url: "https://example.com/products/wf-200"
        }
      }
    },
    build: {
      emptyOutDir: true,
      rollupOptions: {
        input: {
          background: resolve(__dirname, "src/background.ts"),
          contentScript: resolve(__dirname, "src/contentScript.tsx")
        },
        output: {
          entryFileNames: "[name].js",
          chunkFileNames: "assets/[name].js",
          assetFileNames: "assets/[name][extname]"
        }
      }
    },
    define: {
      "import.meta.env.VITE_SALLY_BACKEND_BASE_URL": JSON.stringify(backendBaseUrl)
    }
  };
});
