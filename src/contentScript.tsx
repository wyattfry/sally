import React from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import cssText from "./styles.css?inline";

const HOST_ID = "sally-spec-root";

function mountSally() {
  if (window.name === "sally-signin") {
    return;
  }
  if (document.getElementById(HOST_ID)) {
    return;
  }

  const host = document.createElement("div");
  host.id = HOST_ID;
  document.documentElement.append(host);

  const shadow = host.attachShadow({ mode: "open" });

  const fontStyle = document.createElement("style");
  fontStyle.textContent = `
@font-face {
  font-family: 'DM Sans';
  font-style: normal;
  font-weight: 100 700;
  font-display: swap;
  src: url('${chrome.runtime.getURL("fonts/dm-sans-latin-ext.woff2")}') format('woff2');
  unicode-range: U+0100-02BA, U+02BD-02C5, U+02C7-02CC, U+02CE-02D7, U+02DD-02FF, U+0304, U+0308, U+0329, U+1D00-1DBF, U+1E00-1E9F, U+1EF2-1EFF, U+2020, U+20A0-20AB, U+20AD-20C0, U+2113, U+2C60-2C7F, U+A720-A7FF;
}
@font-face {
  font-family: 'DM Sans';
  font-style: normal;
  font-weight: 100 700;
  font-display: swap;
  src: url('${chrome.runtime.getURL("fonts/dm-sans-latin.woff2")}') format('woff2');
  unicode-range: U+0000-00FF, U+0131, U+0152-0153, U+02BB-02BC, U+02C6, U+02DA, U+02DC, U+0304, U+0308, U+0329, U+2000-206F, U+20AC, U+2122, U+2191, U+2193, U+2212, U+2215, U+FEFF, U+FFFD;
}`;

  const style = document.createElement("style");
  style.textContent = cssText;
  const rootElement = document.createElement("div");
  shadow.append(fontStyle, style, rootElement);

  createRoot(rootElement).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>
  );
}

mountSally();

