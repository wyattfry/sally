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

export {};
