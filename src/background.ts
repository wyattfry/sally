const SPEC_CONTEXT_MENU_ID = "sally-spec-this-page";
const SPEC_MESSAGE = "SALLY_SPEC_THIS_PAGE";

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: SPEC_CONTEXT_MENU_ID,
    title: "SPEC this page",
    contexts: ["page"]
  });
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (info.menuItemId !== SPEC_CONTEXT_MENU_ID || !tab?.id) {
    return;
  }

  chrome.tabs.sendMessage(tab.id, { type: SPEC_MESSAGE });
});

export {};
