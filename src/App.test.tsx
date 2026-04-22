import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";

const storageState: Record<string, unknown> = {};

function installChromeStorageMock() {
  vi.stubGlobal("chrome", {
    storage: {
      local: {
        get: vi.fn(async (keys: string | string[]) => {
          const result: Record<string, unknown> = {};
          for (const key of Array.isArray(keys) ? keys : [keys]) {
            result[key] = storageState[key];
          }
          return result;
        }),
        set: vi.fn(async (values: Record<string, unknown>) => {
          Object.assign(storageState, values);
        }),
        remove: vi.fn(async (key: string) => {
          delete storageState[key];
        })
      }
    }
  });
}

describe("App", () => {
  beforeEach(() => {
    document.title = "Example Co. WF-200 Wall Faucet";
    document.head.innerHTML = `
      <meta property="og:image" content="https://example.com/faucet.jpg" />
      <script type="application/ld+json">
        {"@type":"Product","name":"Wall Faucet","brand":{"name":"Example Co."},"sku":"WF-200"}
      </script>
    `;
    document.body.innerHTML = `
      <p>Polished chrome wall-mounted faucet. Requires rough valve body.</p>
    `;
    for (const key of Object.keys(storageState)) {
      delete storageState[key];
    }
    installChromeStorageMock();
  });

  it("opens Sally, edits a proposal, saves it, and refreshes the item count", async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(await screen.findByText("0 items")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "SPEC" }));

    expect(screen.getByText("Reading page")).toBeInTheDocument();
    expect(await screen.findByDisplayValue("Wall Faucet")).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Zone"), "Primary Bath");
    await user.clear(screen.getByLabelText("Title"));
    await user.type(screen.getByLabelText("Title"), "Wall faucet revised");
    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => expect(screen.queryByText("Sally proposal")).not.toBeInTheDocument());
    expect(await screen.findByText("1 item")).toBeInTheDocument();
  });

  it("does not show undo while creating a new proposal", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByDisplayValue("Wall Faucet")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Undo" })).not.toBeInTheDocument();
  });

  it("supports selecting an existing zone and adding a new zone", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    await screen.findByDisplayValue("Wall Faucet");

    await user.selectOptions(screen.getByLabelText("Zone"), "Primary Bath");
    expect(screen.getByLabelText("Zone")).toHaveValue("Primary Bath");

    await user.selectOptions(screen.getByLabelText("Zone"), "__add_new__");
    await user.type(screen.getByLabelText("New zone"), "Guest Bath");
    await user.click(screen.getByRole("button", { name: "Add zone" }));

    expect(screen.getByLabelText("Zone")).toHaveValue("Guest Bath");
  });

  it("minimizes on Escape and restores the draft without discarding edits", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    const title = await screen.findByLabelText("Title");
    await user.clear(title);
    await user.type(title, "Draft title to keep");

    await user.keyboard("{Escape}");

    expect(screen.queryByLabelText("Sally proposal")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Restore Sally draft" }));

    expect(await screen.findByDisplayValue("Draft title to keep")).toBeInTheDocument();
  });

  it("shows when the current page has already been spec'd", async () => {
    storageState["sally.scheduleItems"] = [
      {
        id: "item-1",
        capturedAt: "2026-04-21T12:00:00.000Z",
        zone: "Primary Bath",
        title: "Wall Faucet",
        manufacturer: "Example Co.",
        modelNumber: "WF-200",
        category: "Faucet",
        description: "Wall-mounted faucet.",
        finish: "Polished Chrome",
        requiredAddOns: [],
        optionalCompanions: [],
        sourceUrl: window.location.href,
        sourceTitle: document.title,
        sourcePdfLinks: []
      }
    ];

    render(<App />);

    expect(await screen.findByText("Page spec'd")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "SPEC" })).toHaveClass("spec-button--specd");
  });

  it("opens a crude schedule viewer from the item count", async () => {
    const user = userEvent.setup();
    storageState["sally.scheduleItems"] = [
      {
        id: "item-1",
        capturedAt: "2026-04-21T12:00:00.000Z",
        zone: "Primary Bath",
        title: "Wall Faucet",
        manufacturer: "Example Co.",
        modelNumber: "WF-200",
        category: "Faucet",
        description: "Wall-mounted faucet.",
        finish: "Polished Chrome",
        requiredAddOns: ["Rough valve body"],
        optionalCompanions: [],
        sourceUrl: "https://example.com/products/wf-200",
        sourceTitle: "Example Co. WF-200 Wall Faucet",
        sourceImageUrl: "https://example.com/faucet.jpg",
        sourcePdfLinks: []
      }
    ];

    render(<App />);

    await user.click(await screen.findByRole("button", { name: /Sally PoC.*1 item/i }));

    expect(screen.getByLabelText("Captured schedule")).toBeInTheDocument();
    expect(screen.getByText("Wall Faucet")).toBeInTheDocument();
    expect(screen.getByText("Primary Bath")).toBeInTheDocument();
    expect(screen.getByText("Example Co.")).toBeInTheDocument();
    expect(screen.getByText("WF-200")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Wall Faucet thumbnail" })).toHaveAttribute(
      "src",
      "https://example.com/faucet.jpg"
    );

    await user.click(screen.getByRole("button", { name: "Close schedule" }));
    expect(screen.queryByLabelText("Captured schedule")).not.toBeInTheDocument();
  });
});
