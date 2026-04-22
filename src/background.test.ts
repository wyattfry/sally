import { beforeEach, describe, expect, it, vi } from "vitest";

describe("background context menu", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it("creates SPEC this page and sends the content script message when clicked", async () => {
    let installedListener: (() => void) | undefined;
    let clickedListener:
      | ((info: { menuItemId: string | number }, tab?: { id?: number }) => void)
      | undefined;
    const create = vi.fn();
    const sendMessage = vi.fn();

    vi.stubGlobal("chrome", {
      runtime: {
        onInstalled: {
          addListener: vi.fn((listener: () => void) => {
            installedListener = listener;
          })
        }
      },
      contextMenus: {
        create,
        onClicked: {
          addListener: vi.fn((listener) => {
            clickedListener = listener;
          })
        }
      },
      tabs: {
        sendMessage
      }
    });

    await import("./background");

    installedListener?.();
    expect(create).toHaveBeenCalledWith({
      id: "sally-spec-this-page",
      title: "SPEC this page",
      contexts: ["page"]
    });

    clickedListener?.({ menuItemId: "sally-spec-this-page" }, { id: 42 });
    expect(sendMessage).toHaveBeenCalledWith(42, { type: "SALLY_SPEC_THIS_PAGE" });
  });
});

