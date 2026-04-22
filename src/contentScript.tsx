import React from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import cssText from "./styles.css?inline";

const HOST_ID = "sally-spec-root";

function mountSally() {
  if (document.getElementById(HOST_ID)) {
    return;
  }

  const host = document.createElement("div");
  host.id = HOST_ID;
  document.documentElement.append(host);

  const shadow = host.attachShadow({ mode: "open" });
  const style = document.createElement("style");
  style.textContent = cssText;
  const rootElement = document.createElement("div");
  shadow.append(style, rootElement);

  createRoot(rootElement).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>
  );
}

mountSally();

