const SPEC_CONTEXT_MENU_ID = "sally-spec-this-page";
const VIEW_CONTEXT_MENU_ID = "sally-view-items";
const SPEC_MESSAGE = "SALLY_SPEC_THIS_PAGE";
const VIEW_MESSAGE = "SALLY_VIEW_ITEMS";

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: SPEC_CONTEXT_MENU_ID,
    title: "SPEC this page",
    contexts: ["page"]
  });
  chrome.contextMenus.create({
    id: VIEW_CONTEXT_MENU_ID,
    title: "View Sally schedule",
    contexts: ["page"]
  });
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (!tab?.id) return;
  if (info.menuItemId === SPEC_CONTEXT_MENU_ID) {
    chrome.tabs.sendMessage(tab.id, { type: SPEC_MESSAGE });
  } else if (info.menuItemId === VIEW_CONTEXT_MENU_ID) {
    chrome.tabs.sendMessage(tab.id, { type: VIEW_MESSAGE });
  }
});

chrome.runtime.onConnect.addListener((port) => {
  if (port.name === "extract-spec") {
    let controller: AbortController | null = null;
    
    port.onMessage.addListener(async (msg) => {
      if (msg.type === "START_EXTRACTION") {
        controller = new AbortController();
        try {
          const response = await fetch(msg.apiUrl, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(msg.request),
            signal: controller.signal
          });

          if (!response.ok) {
            const text = await response.text();
            port.postMessage({ type: "ERROR", kind: "backend", message: text.trim() || "Extraction request failed." });
            return;
          }

          if (!response.body) {
            port.postMessage({ type: "ERROR", kind: "invalid", message: "Empty response from extraction server." });
            return;
          }

          const reader = response.body.getReader();
          const decoder = new TextDecoder();
          let buffer = "";

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });

            const { events, remaining } = parseSSEBuffer(buffer);
            buffer = remaining;

            for (const { event, data } of events) {
              port.postMessage({ type: "SSE_EVENT", event, data });
            }
          }
          port.postMessage({ type: "DONE" });
        } catch (error: any) {
          if (error.name === "AbortError") {
            port.postMessage({ type: "ERROR", kind: "transport", message: "Extraction request timed out." });
          } else if (error instanceof TypeError) {
            port.postMessage({ type: "ERROR", kind: "transport", message: "Extraction backend is unreachable." });
          } else {
            port.postMessage({ type: "ERROR", kind: "transport", message: error.message });
          }
        }
      }
    });

    port.onDisconnect.addListener(() => {
      if (controller) {
        controller.abort();
      }
    });
  }
});

function parseSSEBuffer(buffer: string): {
  events: Array<{ event: string; data: string }>;
  remaining: string;
} {
  const events: Array<{ event: string; data: string }> = [];
  const parts = buffer.split("\n\n");
  const remaining = parts.pop() ?? "";

  for (const part of parts) {
    if (!part.trim()) continue;
    let event = "message";
    let data = "";
    for (const line of part.split("\n")) {
      if (line.startsWith("event: ")) event = line.slice(7).trim();
      else if (line.startsWith("data: ")) data = line.slice(6);
    }
    if (data) events.push({ event, data });
  }

  return { events, remaining };
}

export {};
